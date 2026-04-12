import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:terminal_client/connection/control_client.dart';
import 'package:terminal_client/discovery/mdns_scanner.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;

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
  static const Duration _heartbeatInterval = Duration(seconds: 10);
  static const Duration _reconnectDelayBase = Duration(seconds: 2);
  static const int _reconnectDelayMaxSeconds = 30;

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
  final MdnsScanner _mdnsScanner = MdnsScanner();
  TerminalControlGrpcClient? _client;
  final StreamController<ConnectRequest> _outgoing =
      StreamController<ConnectRequest>();

  StreamSubscription<ConnectResponse>? _incoming;
  String _status = 'Idle';
  int _responses = 0;
  final String _deviceId = 'flutter-${DateTime.now().millisecondsSinceEpoch}';
  uiv1.Node? _activeRoot;
  String _lastNotification = '';
  final TextEditingController _terminalInputController =
      TextEditingController();
  bool _isScanning = false;
  List<DiscoveredServer> _discoveredServers = [];
  String? _selectedDiscoveredServer;
  Timer? _heartbeatTimer;
  Timer? _reconnectTimer;
  bool _shouldStayConnected = false;
  bool _isConnecting = false;
  int _reconnectAttempt = 0;

  Future<void> _startStream({bool userInitiated = true}) async {
    if (_isConnecting) {
      return;
    }
    if (userInitiated) {
      _shouldStayConnected = true;
      _reconnectAttempt = 0;
      _cancelReconnectTimer();
    }

    final host = _hostController.text.trim();
    final port = int.tryParse(_portController.text.trim());
    final mediaQuery = MediaQuery.of(context);
    final size = mediaQuery.size;
    if (host.isEmpty || port == null || port <= 0 || port > 65535) {
      _shouldStayConnected = false;
      setState(() {
        _status = 'Invalid host or port';
      });
      return;
    }

    _isConnecting = true;
    if (mounted) {
      setState(() {
        _status = userInitiated ? 'Connecting...' : 'Reconnecting...';
      });
    }
    try {
      await _incoming?.cancel();
      _incoming = null;
      final existingClient = _client;
      if (existingClient != null) {
        await existingClient.shutdown();
      }
      _client = TerminalControlGrpcClient(host: host, port: port);

      final stream = _client!.connect(_outgoing.stream);
      _incoming = stream.listen(
        (ConnectResponse response) {
          if (!mounted) {
            return;
          }
          _reconnectAttempt = 0;
          setState(() {
            _responses += 1;
            _status = _statusFromResponse(response);
            if (response.hasSetUi() && response.setUi.hasRoot()) {
              _activeRoot = response.setUi.root;
            }
            if (response.hasNotification()) {
              _lastNotification = response.notification.body;
            }
            if (response.hasCommandResult() &&
                response.commandResult.notification.isNotEmpty) {
              _lastNotification = response.commandResult.notification;
            }
            if (response.hasError()) {
              _lastNotification = response.error.message;
            }
          });
        },
        onError: (Object error) {
          _handleStreamClosed('Stream error: $error');
        },
        onDone: () {
          _handleStreamClosed('Disconnected');
        },
      );

      _startHeartbeatLoop();
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
    } catch (error) {
      _handleStreamClosed('Connection error: $error');
    } finally {
      _isConnecting = false;
    }
  }

  void _startHeartbeatLoop() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = Timer.periodic(_heartbeatInterval, (_) {
      if (!_shouldStayConnected || _deviceId.isEmpty) {
        return;
      }
      _outgoing.add(
        TerminalControlGrpcClient.heartbeatRequest(
          deviceId: _deviceId,
          unixMs: DateTime.now().millisecondsSinceEpoch,
        ),
      );
    });
  }

  void _stopHeartbeatLoop() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = null;
  }

  void _cancelReconnectTimer() {
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
  }

  void _handleStreamClosed(String status) {
    _stopHeartbeatLoop();
    _incoming = null;
    final existingClient = _client;
    _client = null;
    if (existingClient != null) {
      unawaited(existingClient.shutdown());
    }
    if (mounted) {
      setState(() {
        _status = status;
      });
    }
    _scheduleReconnect();
  }

  void _scheduleReconnect() {
    if (!_shouldStayConnected || _isConnecting) {
      return;
    }
    if (_reconnectTimer?.isActive ?? false) {
      return;
    }
    _reconnectAttempt += 1;
    final scaledSeconds = _reconnectDelayBase.inSeconds *
        math.pow(2, _reconnectAttempt - 1).toInt();
    final delaySeconds = math.min(_reconnectDelayMaxSeconds, scaledSeconds);
    if (mounted) {
      setState(() {
        _status = 'Connection lost, retrying in ${delaySeconds}s...';
      });
    }
    _reconnectTimer = Timer(Duration(seconds: delaySeconds), () {
      _reconnectTimer = null;
      if (!_shouldStayConnected || !mounted) {
        return;
      }
      unawaited(_startStream(userInitiated: false));
    });
  }

  Future<void> _scanForServers() async {
    if (_isScanning) {
      return;
    }
    setState(() {
      _isScanning = true;
      _status = 'Scanning LAN for server...';
    });
    try {
      final found = await _mdnsScanner.scan();
      if (!mounted) {
        return;
      }
      setState(() {
        _discoveredServers = found;
        if (found.isNotEmpty) {
          _selectedDiscoveredServer = '${found.first.host}:${found.first.port}';
          _hostController.text = found.first.host;
          _portController.text = found.first.port.toString();
          _status = 'Found ${found.length} server(s)';
        } else {
          _selectedDiscoveredServer = null;
          _status = 'No servers discovered';
        }
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _status = 'Discovery error: $error';
      });
    } finally {
      if (mounted) {
        setState(() {
          _isScanning = false;
        });
      }
    }
  }

  Future<void> _stopStream() async {
    _shouldStayConnected = false;
    _reconnectAttempt = 0;
    _cancelReconnectTimer();
    _stopHeartbeatLoop();
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
        _activeRoot = null;
      });
    }
  }

  @override
  void dispose() {
    _shouldStayConnected = false;
    _cancelReconnectTimer();
    _stopHeartbeatLoop();
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
    _terminalInputController.dispose();
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
              Row(
                children: [
                  ElevatedButton(
                    onPressed: _isScanning ? null : _scanForServers,
                    child: Text(_isScanning ? 'Scanning...' : 'Scan LAN'),
                  ),
                ],
              ),
              if (_discoveredServers.isNotEmpty) ...[
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  key: ValueKey<String?>(_selectedDiscoveredServer),
                  initialValue: _selectedDiscoveredServer,
                  decoration:
                      const InputDecoration(labelText: 'Discovered Server'),
                  items: _discoveredServers
                      .map(
                        (server) => DropdownMenuItem<String>(
                          value: '${server.host}:${server.port}',
                          child: Text(
                            '${server.name} (${server.host}:${server.port})',
                          ),
                        ),
                      )
                      .toList(),
                  onChanged: (value) {
                    if (value == null) {
                      return;
                    }
                    final parts = value.split(':');
                    if (parts.length != 2) {
                      return;
                    }
                    setState(() {
                      _selectedDiscoveredServer = value;
                      _hostController.text = parts[0];
                      _portController.text = parts[1];
                    });
                  },
                ),
              ],
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
              if (_lastNotification.isNotEmpty) ...[
                const SizedBox(height: 12),
                Text('Notification: $_lastNotification'),
              ],
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
              if (_activeRoot != null) ...[
                const SizedBox(height: 24),
                const Divider(),
                const SizedBox(height: 12),
                SizedBox(
                  height: 320,
                  child: _renderNode(_activeRoot!),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  String _statusFromResponse(ConnectResponse response) {
    if (response.hasError()) {
      return 'Server error';
    }
    if (response.hasRegisterAck()) {
      return 'Registered';
    }
    if (response.hasCommandResult()) {
      return 'Command response';
    }
    if (response.hasSetUi()) {
      return 'UI updated';
    }
    return 'Connected';
  }

  Future<void> _sendUiAction({
    required String componentId,
    required String action,
    required String value,
  }) async {
    if (_deviceId.isEmpty) {
      return;
    }
    _outgoing.add(
      ConnectRequest()
        ..input = (iov1.InputEvent()
          ..deviceId = _deviceId
          ..uiAction = (iov1.UIAction()
            ..componentId = componentId
            ..action = action
            ..value = value)),
    );
  }

  String _nodeId(uiv1.Node node) {
    if (node.id.isNotEmpty) {
      return node.id;
    }
    return node.props['id'] ?? '';
  }

  Widget _renderNode(uiv1.Node node) {
    switch (node.whichWidget()) {
      case uiv1.Node_Widget.stack:
        return Container(
          color: _parseHexColor(node.props['background']),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: node.children.map(_renderNode).toList(),
          ),
        );
      case uiv1.Node_Widget.row:
        return Row(children: node.children.map(_renderNode).toList());
      case uiv1.Node_Widget.scroll:
        return SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: node.children.map(_renderNode).toList(),
          ),
        );
      case uiv1.Node_Widget.text:
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 4),
          child: SelectableText(
            node.text.value,
            style: TextStyle(
              color: _parseHexColor(node.text.color),
              fontFamily: node.text.style == 'monospace' ? 'monospace' : null,
            ),
          ),
        );
      case uiv1.Node_Widget.textInput:
        final componentId = _nodeId(node);
        return TextField(
          controller: _terminalInputController,
          decoration: InputDecoration(
            hintText: node.textInput.placeholder,
          ),
          autofocus: node.textInput.autofocus,
          onSubmitted: (value) async {
            await _sendUiAction(
              componentId: componentId.isNotEmpty ? componentId : 'text_input',
              action: 'submit',
              value: value,
            );
            _terminalInputController.clear();
          },
        );
      case uiv1.Node_Widget.button:
        final componentId = _nodeId(node);
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 4),
          child: ElevatedButton(
            onPressed: () {
              unawaited(
                _sendUiAction(
                  componentId: componentId.isNotEmpty ? componentId : 'button',
                  action: node.button.action.isNotEmpty
                      ? node.button.action
                      : 'tap',
                  value: '',
                ),
              );
            },
            child: Text(node.button.label),
          ),
        );
      case uiv1.Node_Widget.notSet:
        break;
      default:
        if (node.children.isEmpty) {
          return const SizedBox.shrink();
        }
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: node.children.map(_renderNode).toList(),
        );
    }
    return const SizedBox.shrink();
  }

  Color? _parseHexColor(String? raw) {
    if (raw == null || raw.isEmpty) {
      return null;
    }
    var value = raw.trim();
    if (value.startsWith('#')) {
      value = value.substring(1);
    }
    if (value.length == 6) {
      value = 'FF$value';
    }
    if (value.length != 8) {
      return null;
    }
    final parsed = int.tryParse(value, radix: 16);
    if (parsed == null) {
      return null;
    }
    return Color(parsed);
  }
}
