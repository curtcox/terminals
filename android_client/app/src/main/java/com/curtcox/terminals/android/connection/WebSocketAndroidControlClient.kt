package com.curtcox.terminals.android.connection

import android.util.Base64
import com.google.protobuf.InvalidProtocolBufferException
import java.io.BufferedInputStream
import java.io.BufferedOutputStream
import java.io.Closeable
import java.io.EOFException
import java.io.IOException
import java.net.Socket
import java.security.MessageDigest
import java.security.SecureRandom
import javax.net.ssl.SSLSocketFactory
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import terminals.control.v1.Control

private const val WebSocketGuid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
internal const val BinaryOpcode = 0x2
private const val CloseOpcode = 0x8
private const val PingOpcode = 0x9
internal const val PongOpcode = 0xA

class WebSocketAndroidControlClient(
    private val deviceId: String,
    private val responseSink: AndroidControlResponseSink? = null,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.IO),
) : AndroidControlClient {
    private val random = SecureRandom()
    private var socket: Socket? = null
    private var input: BufferedInputStream? = null
    private var output: BufferedOutputStream? = null
    private var readJob: Job? = null
    private var sequence: Long = 0
    private var sessionId: String = ""

    @Volatile
    private var closingSocket: Boolean = false

    override suspend fun connect(endpoint: EndpointResolution) = withContext(Dispatchers.IO) {
        close()

        val connectedSocket = if (endpoint.secure) {
            SSLSocketFactory.getDefault().createSocket(endpoint.host, endpoint.port) as Socket
        } else {
            Socket(endpoint.host, endpoint.port)
        }
        connectedSocket.tcpNoDelay = true
        val nextInput = BufferedInputStream(connectedSocket.getInputStream())
        val nextOutput = BufferedOutputStream(connectedSocket.getOutputStream())

        val path = endpoint.path.ifBlank { "/control" }
        val key = randomWebSocketKey()
        writeUpgradeRequest(nextOutput, endpoint, path, key)
        verifyUpgradeResponse(nextInput, key)

        socket = connectedSocket
        input = nextInput
        output = nextOutput
        sequence = 0
        sessionId = ""

        writeEnvelope(
            Control.WireEnvelope.newBuilder()
                .setProtocolVersion(AndroidWireProtocolVersion)
                .setTransportHello(
                    Control.TransportHello.newBuilder()
                        .setProtocolVersion(AndroidWireProtocolVersion)
                        .setDesiredDeviceId(deviceId)
                        .addSupportedCarriers(Control.CarrierKind.CARRIER_KIND_WEBSOCKET),
                )
                .build(),
        )
        val ack = readEnvelope()
        if (!ack.hasTransportHelloAck()) {
            throw IOException("websocket transport hello was not acknowledged")
        }
        sessionId = ack.transportHelloAck.sessionId
        closingSocket = false
        readJob = scope.launch { readResponses() }
    }

    override suspend fun send(request: Control.ConnectRequest) = withContext(Dispatchers.IO) {
        ensureConnected()
        sequence += 1
        writeEnvelope(
            Control.WireEnvelope.newBuilder()
                .setProtocolVersion(AndroidWireProtocolVersion)
                .setSessionId(sessionId)
                .setSequence(sequence)
                .setClientMessage(request)
                .build(),
        )
    }

    override suspend fun close() = withContext(Dispatchers.IO) {
        closingSocket = true
        readJob?.cancel()
        readJob = null
        closeQuietly(input)
        closeQuietly(output)
        closeQuietly(socket)
        input = null
        output = null
        socket = null
        sessionId = ""
        sequence = 0
        closingSocket = false
    }

    private suspend fun readResponses() {
        try {
            while (true) {
                val envelope = withContext(Dispatchers.IO) { readEnvelope() }
                when {
                    envelope.hasServerMessage() -> responseSink?.onResponse(envelope.serverMessage)
                    envelope.hasTransportError() -> throw IOException(
                        "transport error ${envelope.transportError.code}: ${envelope.transportError.message}",
                    )
                }
            }
        } catch (cancelled: CancellationException) {
            throw cancelled
        } catch (error: Throwable) {
            if (!closingSocket) {
                responseSink?.onTransportTerminated(error)
            }
        }
    }

    private fun writeUpgradeRequest(
        output: BufferedOutputStream,
        endpoint: EndpointResolution,
        path: String,
        key: String,
    ) {
        val host = if (endpoint.port == 80 || endpoint.port == 443) endpoint.host else "${endpoint.host}:${endpoint.port}"
        val request = buildString {
            append("GET $path HTTP/1.1\r\n")
            append("Host: $host\r\n")
            append("Upgrade: websocket\r\n")
            append("Connection: Upgrade\r\n")
            append("Sec-WebSocket-Key: $key\r\n")
            append("Sec-WebSocket-Version: 13\r\n")
            append("Origin: ${if (endpoint.secure) "https" else "http"}://$host\r\n")
            append("\r\n")
        }
        output.write(request.toByteArray(Charsets.US_ASCII))
        output.flush()
    }

    private fun verifyUpgradeResponse(input: BufferedInputStream, key: String) {
        val headers = readHttpHeaders(input)
        val status = headers.firstOrNull() ?: throw IOException("empty websocket upgrade response")
        if (!status.contains(" 101 ")) throw IOException("websocket upgrade failed: $status")
        val accept = headers
            .drop(1)
            .firstOrNull { it.startsWith("Sec-WebSocket-Accept:", ignoreCase = true) }
            ?.substringAfter(':')
            ?.trim()
        if (accept != expectedAccept(key)) throw IOException("websocket upgrade accept mismatch")
    }

    private fun readHttpHeaders(input: BufferedInputStream): List<String> {
        val bytes = mutableListOf<Byte>()
        while (true) {
            val next = input.read()
            if (next < 0) throw EOFException("websocket upgrade response ended early")
            bytes += next.toByte()
            if (bytes.size >= 4 &&
                bytes[bytes.size - 4] == '\r'.code.toByte() &&
                bytes[bytes.size - 3] == '\n'.code.toByte() &&
                bytes[bytes.size - 2] == '\r'.code.toByte() &&
                bytes[bytes.size - 1] == '\n'.code.toByte()
            ) {
                return bytes.toByteArray().toString(Charsets.US_ASCII).trimEnd().lines()
            }
        }
    }

    private fun writeEnvelope(envelope: Control.WireEnvelope) {
        val target = output ?: throw IOException("websocket is not connected")
        writeFrame(target, envelope.toByteArray(), masked = true)
    }

    private fun readEnvelope(): Control.WireEnvelope {
        val source = input ?: throw IOException("websocket is not connected")
        while (true) {
            val frame = readFrame(source)
            when (frame.opcode) {
                BinaryOpcode -> return try {
                    Control.WireEnvelope.parseFrom(frame.payload)
                } catch (error: InvalidProtocolBufferException) {
                    throw IOException("decode websocket envelope", error)
                }
                PingOpcode -> output?.let { writeFrame(it, frame.payload, masked = true, opcode = PongOpcode) }
                CloseOpcode -> throw EOFException("websocket closed")
            }
        }
    }

    private fun ensureConnected() {
        if (socket == null || output == null) throw IOException("websocket is not connected")
    }

    private fun randomWebSocketKey(): String {
        val bytes = ByteArray(16)
        random.nextBytes(bytes)
        return Base64.encodeToString(bytes, Base64.NO_WRAP)
    }

    private fun expectedAccept(key: String): String {
        val digest = MessageDigest.getInstance("SHA-1")
            .digest((key + WebSocketGuid).toByteArray(Charsets.US_ASCII))
        return Base64.encodeToString(digest, Base64.NO_WRAP)
    }

    private fun closeQuietly(closeable: Closeable?) {
        runCatching { closeable?.close() }
    }

    private fun closeQuietly(socket: Socket?) {
        runCatching { socket?.close() }
    }
}

internal data class WebSocketFrame(val opcode: Int, val payload: ByteArray)

internal fun writeFrame(
    output: BufferedOutputStream,
    payload: ByteArray,
    masked: Boolean,
    opcode: Int = BinaryOpcode,
) {
    output.write(0x80 or opcode)
    val maskBit = if (masked) 0x80 else 0
    when {
        payload.size < 126 -> output.write(maskBit or payload.size)
        payload.size <= 0xFFFF -> {
            output.write(maskBit or 126)
            output.write((payload.size ushr 8) and 0xFF)
            output.write(payload.size and 0xFF)
        }
        else -> {
            output.write(maskBit or 127)
            val size = payload.size.toLong()
            for (shift in 56 downTo 0 step 8) {
                output.write(((size ushr shift) and 0xFF).toInt())
            }
        }
    }
    val bytes = if (masked) {
        val mask = ByteArray(4)
        SecureRandom().nextBytes(mask)
        output.write(mask)
        payload.mapIndexed { index, byte -> (byte.toInt() xor mask[index % 4].toInt()).toByte() }.toByteArray()
    } else {
        payload
    }
    output.write(bytes)
    output.flush()
}

internal fun readFrame(input: BufferedInputStream): WebSocketFrame {
    val first = input.read()
    if (first < 0) throw EOFException("websocket frame ended before header")
    val second = input.read()
    if (second < 0) throw EOFException("websocket frame ended before length")
    val fin = first and 0x80 != 0
    if (!fin) throw IOException("fragmented websocket frames are not supported")
    val opcode = first and 0x0F
    val masked = second and 0x80 != 0
    val lengthCode = second and 0x7F
    val length = when (lengthCode) {
        in 0..125 -> lengthCode.toLong()
        126 -> ((readRequired(input) shl 8) or readRequired(input)).toLong()
        127 -> {
            var size = 0L
            repeat(8) { size = (size shl 8) or readRequired(input).toLong() }
            size
        }
        else -> throw IOException("invalid websocket frame length")
    }
    if (length > Int.MAX_VALUE) throw IOException("websocket frame too large: $length")
    val mask = if (masked) ByteArray(4).also { input.readFully(it) } else null
    val payload = ByteArray(length.toInt()).also { input.readFully(it) }
    if (mask != null) {
        for (index in payload.indices) {
            payload[index] = (payload[index].toInt() xor mask[index % 4].toInt()).toByte()
        }
    }
    return WebSocketFrame(opcode, payload)
}

private fun readRequired(input: BufferedInputStream): Int {
    val next = input.read()
    if (next < 0) throw EOFException("websocket frame ended early")
    return next
}

private fun BufferedInputStream.readFully(target: ByteArray) {
    var offset = 0
    while (offset < target.size) {
        val read = read(target, offset, target.size - offset)
        if (read < 0) throw EOFException("websocket frame payload ended early")
        offset += read
    }
}
