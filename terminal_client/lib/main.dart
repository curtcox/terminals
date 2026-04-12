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
  final TerminalControlGrpcClient _client = TerminalControlGrpcClient(
    host: '127.0.0.1',
    port: 50051,
  );
  final StreamController<ConnectRequest> _outgoing =
      StreamController<ConnectRequest>();

  StreamSubscription<ConnectResponse>? _incoming;
  String _status = 'Idle';
  int _responses = 0;

  Future<void> _startStream() async {
    await _incoming?.cancel();

    final stream = _client.connect(_outgoing.stream);
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

    _outgoing.add(
      TerminalControlGrpcClient.registerRequest(
        deviceId: 'flutter-client-1',
        deviceName: 'Flutter Client',
        deviceType: 'desktop',
        platform: 'flutter',
      ),
    );
    _outgoing.add(
      TerminalControlGrpcClient.heartbeatRequest(
        deviceId: 'flutter-client-1',
        unixMs: DateTime.now().millisecondsSinceEpoch,
      ),
    );
  }

  Future<void> _stopStream() async {
    await _incoming?.cancel();
    _incoming = null;
    if (mounted) {
      setState(() {
        _status = 'Disconnected';
      });
    }
  }

  @override
  void dispose() {
    _incoming?.cancel();
    _outgoing.close();
    _client.shutdown();
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
