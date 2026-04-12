import 'dart:async';

import 'package:flutter/material.dart';
import 'package:terminal_client/connection/control_client.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';

void main() {
  runApp(const TerminalClientApp());
}

class TerminalClientApp extends StatelessWidget {
  const TerminalClientApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Terminal Client',
      home: const _ControlStreamScaffold(),
    );
  }
}

class _ControlStreamScaffold extends StatefulWidget {
  const _ControlStreamScaffold();

  @override
  State<_ControlStreamScaffold> createState() => _ControlStreamScaffoldState();
}

class _ControlStreamScaffoldState extends State<_ControlStreamScaffold> {
  final TextEditingController _hostController = TextEditingController(
    text: '127.0.0.1',
  );
  final TextEditingController _portController = TextEditingController(
    text: '50051',
  );
  final TextEditingController _deviceNameController = TextEditingController(
    text: 'Flutter Client',
  );
  final TextEditingController _deviceTypeController = TextEditingController(
    text: 'desktop',
  );
  final TextEditingController _platformController = TextEditingController(
    text: 'flutter',
  );
  TerminalControlGrpcClient? _client;
  final StreamController<ConnectRequest> _outgoing =
      StreamController<ConnectRequest>();

  StreamSubscription<ConnectResponse>? _incoming;
  String _status = 'Idle';
  int _responses = 0;
  String _deviceId = 'flutter-client-1';

  Future<void> _startStream() async {
    final host = _hostController.text.trim();
    final port = int.tryParse(_portController.text.trim());
    if (host.isEmpty || port == null || port <= 0 || port > 65535) {
      setState(() {
        _status = 'Invalid host or port';
      });
      return;
    }

    await _incoming?.cancel();
    final existingClient = _client;
    if (existingClient != null) {
      await existingClient.shutdown();
    }
    _client = TerminalControlGrpcClient(host: host, port: port);

    final stream = _client!.connect(_outgoing.stream);
    _incoming = stream.listen(
      (_) {
        if (!mounted) {
          return;
        }
        setState(() {
          _responses += 1;
          _status = 'Connected';
        });
      },
      onError: (Object error) {
        if (!mounted) {
          return;
        }
        setState(() {
          _status = 'Stream error: $error';
        });
      },
      onDone: () {
        if (!mounted) {
          return;
        }
        setState(() {
          _status = 'Disconnected';
        });
      },
    );

    _deviceId = 'flutter-${DateTime.now().millisecondsSinceEpoch}';
    final mediaQuery = MediaQuery.of(context);
    final size = mediaQuery.size;
    _outgoing.add(
      TerminalControlGrpcClient.registerRequest(
        deviceId: _deviceId,
        deviceName: _deviceNameController.text.trim(),
        deviceType: _deviceTypeController.text.trim(),
        platform: _platformController.text.trim(),
        screenWidth: size.width.round(),
        screenHeight: size.height.round(),
        screenDensity: mediaQuery.devicePixelRatio,
        screenTouch: true,
      ),
    );
    _outgoing.add(
      TerminalControlGrpcClient.heartbeatRequest(
        deviceId: _deviceId,
        unixMs: DateTime.now().millisecondsSinceEpoch,
      ),
    );
  }

  Future<void> _stopStream() async {
    await _incoming?.cancel();
    _incoming = null;
    final existingClient = _client;
    if (existingClient != null) {
      await existingClient.shutdown();
      _client = null;
    }
    if (mounted) {
      setState(() {
        _status = 'Disconnected';
      });
    }
  }

  @override
  void dispose() {
    final incoming = _incoming;
    if (incoming != null) {
      unawaited(incoming.cancel());
    }
    _outgoing.close();
    final existingClient = _client;
    if (existingClient != null) {
      unawaited(existingClient.shutdown());
    }
    _hostController.dispose();
    _portController.dispose();
    _deviceNameController.dispose();
    _deviceTypeController.dispose();
    _platformController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: _hostController,
                decoration: const InputDecoration(
                  labelText: 'Server Host',
                  hintText: '127.0.0.1',
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _portController,
                decoration: const InputDecoration(labelText: 'Server Port'),
                keyboardType: TextInputType.number,
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _deviceNameController,
                decoration: const InputDecoration(labelText: 'Device Name'),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _deviceTypeController,
                decoration: const InputDecoration(labelText: 'Device Type'),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _platformController,
                decoration: const InputDecoration(labelText: 'Platform'),
              ),
              const SizedBox(height: 20),
              Text('Control Stream: $_status'),
              const SizedBox(height: 12),
              Text('Responses: $_responses'),
              const SizedBox(height: 20),
              Wrap(
                spacing: 12,
                children: [
                  ElevatedButton(
                    onPressed: _startStream,
                    child: const Text('Connect Stream'),
                  ),
                  OutlinedButton(
                    onPressed: _stopStream,
                    child: const Text('Disconnect'),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}
