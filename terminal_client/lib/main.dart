import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:terminal_client/connection/control_client.dart';
import 'package:terminal_client/discovery/mdns_scanner.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;

typedef TerminalControlClientFactory = TerminalControlClient Function({
  required String host,
  required int port,
});

Duration calculateReconnectDelay({
  required int reconnectAttempt,
  required Duration reconnectDelayBase,
  required int reconnectDelayMaxSeconds,
}) {
  final scaledMs = reconnectDelayBase.inMilliseconds *
      math.pow(2, reconnectAttempt - 1).toInt();
  final maxMs = reconnectDelayMaxSeconds * 1000;
  final delayMs = math.min(maxMs, math.max(1, scaledMs));
  return Duration(milliseconds: delayMs);
}

void main() {
  runApp(const TerminalClientApp());
}

class TerminalClientApp extends StatelessWidget {
  const TerminalClientApp({
    super.key,
    this.clientFactory = TerminalControlGrpcClient.new,
    this.heartbeatInterval = const Duration(seconds: 10),
    this.reconnectDelayBase = const Duration(seconds: 2),
    this.reconnectDelayMaxSeconds = 30,
  });

  final TerminalControlClientFactory clientFactory;
  final Duration heartbeatInterval;
  final Duration reconnectDelayBase;
  final int reconnectDelayMaxSeconds;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Terminal Client',
      home: _ControlStreamScaffold(
        clientFactory: clientFactory,
        heartbeatInterval: heartbeatInterval,
        reconnectDelayBase: reconnectDelayBase,
        reconnectDelayMaxSeconds: reconnectDelayMaxSeconds,
      ),
    );
  }
}

class _ControlStreamScaffold extends StatefulWidget {
  const _ControlStreamScaffold({
    required this.clientFactory,
    required this.heartbeatInterval,
    required this.reconnectDelayBase,
    required this.reconnectDelayMaxSeconds,
  });

  final TerminalControlClientFactory clientFactory;
  final Duration heartbeatInterval;
  final Duration reconnectDelayBase;
  final int reconnectDelayMaxSeconds;

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
  final MdnsScanner _mdnsScanner = MdnsScanner();
  TerminalControlClient? _client;
  final StreamController<ConnectRequest> _outgoing =
      StreamController<ConnectRequest>.broadcast();

  StreamSubscription<ConnectResponse>? _incoming;
  String _status = 'Idle';
  int _responses = 0;
  final String _deviceId = 'flutter-${DateTime.now().millisecondsSinceEpoch}';
  uiv1.Node? _activeRoot;
  int _activeRootRevision = 0;
  String _activeTransition = 'none';
  Duration _activeTransitionDuration = Duration.zero;
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
      _client = widget.clientFactory(host: host, port: port);

      final stream = _client!.connect(_outgoing.stream);
      _incoming = stream.listen(
        (ConnectResponse response) {
          if (!mounted) {
            return;
          }
          _reconnectAttempt = 0;
          setState(() {
            _responses += 1;
            final responseStatus = _statusFromResponse(response);
            if (responseStatus.isNotEmpty) {
              _status = responseStatus;
            }
            var nextRoot = _activeRoot;
            var uiChanged = false;
            if (response.hasSetUi() && response.setUi.hasRoot()) {
              nextRoot = response.setUi.root.deepCopy();
              uiChanged = true;
            }
            if (response.hasUpdateUi()) {
              final updatedRoot = _applyUpdateUi(
                currentRoot: nextRoot,
                update: response.updateUi,
              );
              if (!identical(updatedRoot, nextRoot)) {
                uiChanged = true;
              }
              nextRoot = updatedRoot;
            }
            if (response.hasTransitionUi()) {
              _applyTransitionHint(response.transitionUi);
              if (nextRoot != null) {
                uiChanged = true;
              }
            }
            _activeRoot = nextRoot;
            if (uiChanged) {
              _activeRootRevision += 1;
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
          unawaited(_handleStreamClosed('Stream error: $error'));
        },
        onDone: () {
          unawaited(_handleStreamClosed('Disconnected'));
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
      await _handleStreamClosed('Connection error: $error');
    } finally {
      _isConnecting = false;
    }
  }

  void _startHeartbeatLoop() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = Timer.periodic(widget.heartbeatInterval, (_) {
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

  Future<void> _handleStreamClosed(String status) async {
    _stopHeartbeatLoop();
    _incoming = null;
    final existingClient = _client;
    _client = null;
    if (existingClient != null) {
      await existingClient.shutdown();
    }
    if (mounted) {
      setState(() {
        _status = status;
      });
    }
    _scheduleReconnect();
  }

  void _scheduleReconnect() {
    if (!_shouldStayConnected) {
      return;
    }
    if (_isConnecting) {
      if (_reconnectTimer?.isActive ?? false) {
        return;
      }
      _reconnectTimer = Timer(const Duration(milliseconds: 50), () {
        _reconnectTimer = null;
        _scheduleReconnect();
      });
      return;
    }
    if (_reconnectTimer?.isActive ?? false) {
      return;
    }
    _reconnectAttempt += 1;
    final reconnectDelay = calculateReconnectDelay(
      reconnectAttempt: _reconnectAttempt,
      reconnectDelayBase: widget.reconnectDelayBase,
      reconnectDelayMaxSeconds: widget.reconnectDelayMaxSeconds,
    );
    final displaySeconds = reconnectDelay.inMilliseconds <= 1000
        ? 1
        : (reconnectDelay.inMilliseconds / 1000).ceil();
    if (mounted) {
      setState(() {
        _status = 'Connection lost, retrying in ${displaySeconds}s...';
      });
    }
    _reconnectTimer = Timer(reconnectDelay, () {
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
                  child: AnimatedSwitcher(
                    duration: _activeTransitionDuration,
                    switchInCurve: Curves.easeOut,
                    switchOutCurve: Curves.easeIn,
                    transitionBuilder: _buildTransition,
                    child: KeyedSubtree(
                      key: ValueKey<int>(_activeRootRevision),
                      child: _renderNode(_activeRoot!),
                    ),
                  ),
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
    if (response.hasTransitionUi()) {
      return 'UI transition';
    }
    if (response.hasUpdateUi()) {
      return 'UI patched';
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

  void _applyTransitionHint(uiv1.TransitionUI transitionUi) {
    final transition = transitionUi.transition.trim().toLowerCase();
    final hasTransition = transition.isNotEmpty && transition != 'none';
    final defaultDuration = hasTransition ? 250 : 0;
    final durationMs =
        transitionUi.durationMs > 0 ? transitionUi.durationMs : defaultDuration;
    _activeTransition = transition;
    _activeTransitionDuration = Duration(milliseconds: durationMs);
    _lastNotification =
        'Transition: ${transitionUi.transition} (${transitionUi.durationMs}ms)';
  }

  Widget _buildTransition(Widget child, Animation<double> animation) {
    switch (_activeTransition) {
      case 'fade':
        return FadeTransition(opacity: animation, child: child);
      case 'scale':
        return ScaleTransition(scale: animation, child: child);
      case 'slide':
      case 'slide_left':
      case 'slide-left':
        return _buildSlideTransition(
          child: child,
          animation: animation,
          beginOffset: const Offset(0.15, 0),
        );
      case 'slide_right':
      case 'slide-right':
        return _buildSlideTransition(
          child: child,
          animation: animation,
          beginOffset: const Offset(-0.15, 0),
        );
      case 'slide_up':
      case 'slide-up':
        return _buildSlideTransition(
          child: child,
          animation: animation,
          beginOffset: const Offset(0, 0.15),
        );
      case 'slide_down':
      case 'slide-down':
        return _buildSlideTransition(
          child: child,
          animation: animation,
          beginOffset: const Offset(0, -0.15),
        );
      default:
        return child;
    }
  }

  Widget _buildSlideTransition({
    required Widget child,
    required Animation<double> animation,
    required Offset beginOffset,
  }) {
    return SlideTransition(
      position: Tween<Offset>(
        begin: beginOffset,
        end: Offset.zero,
      ).animate(animation),
      child: child,
    );
  }

  uiv1.Node? _applyUpdateUi({
    required uiv1.Node? currentRoot,
    required uiv1.UpdateUI update,
  }) {
    if (!update.hasNode()) {
      return currentRoot;
    }
    final targetID = update.componentId.trim();
    final replacement = update.node.deepCopy();
    if (targetID.isEmpty) {
      return replacement;
    }
    if (currentRoot == null) {
      return null;
    }

    final root = currentRoot.deepCopy();
    if (_nodeId(root) == targetID) {
      return replacement;
    }
    final replaced = _replaceNodeByID(
      current: root,
      targetID: targetID,
      replacement: replacement,
    );
    if (!replaced) {
      return currentRoot;
    }
    return root;
  }

  bool _replaceNodeByID({
    required uiv1.Node current,
    required String targetID,
    required uiv1.Node replacement,
  }) {
    for (var i = 0; i < current.children.length; i++) {
      final child = current.children[i];
      if (_nodeId(child) == targetID) {
        current.children[i] = replacement.deepCopy();
        return true;
      }
      if (_replaceNodeByID(
        current: child,
        targetID: targetID,
        replacement: replacement,
      )) {
        return true;
      }
    }
    return false;
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
      case uiv1.Node_Widget.grid:
        final columns = node.grid.columns > 0 ? node.grid.columns : 1;
        return LayoutBuilder(
          builder: (context, constraints) {
            const spacing = 8.0;
            final maxWidth = constraints.maxWidth.isFinite
                ? constraints.maxWidth
                : MediaQuery.of(context).size.width;
            final totalSpacing = spacing * (columns - 1);
            final itemWidth =
                columns <= 1 ? maxWidth : (maxWidth - totalSpacing) / columns;
            return Wrap(
              spacing: spacing,
              runSpacing: spacing,
              children: node.children
                  .map(
                    (child) => SizedBox(
                      width: itemWidth,
                      child: _renderNode(child),
                    ),
                  )
                  .toList(),
            );
          },
        );
      case uiv1.Node_Widget.scroll:
        return SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: node.children.map(_renderNode).toList(),
          ),
        );
      case uiv1.Node_Widget.padding:
        return Padding(
          padding: EdgeInsets.all(node.padding.all.toDouble()),
          child: _renderNodeChildren(node.children),
        );
      case uiv1.Node_Widget.center:
        return Center(child: _renderNodeChildren(node.children));
      case uiv1.Node_Widget.expand:
        return Expanded(child: _renderNodeChildren(node.children));
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
      case uiv1.Node_Widget.slider:
        final componentId = _nodeId(node);
        final min = node.slider.min;
        final max = node.slider.max > min ? node.slider.max : min + 1;
        final value = node.slider.value.clamp(min, max).toDouble();
        return Slider(
          value: value,
          min: min,
          max: max,
          onChanged: (nextValue) {
            unawaited(
              _sendUiAction(
                componentId: componentId.isNotEmpty ? componentId : 'slider',
                action: 'change',
                value: nextValue.toString(),
              ),
            );
          },
        );
      case uiv1.Node_Widget.toggle:
        final componentId = _nodeId(node);
        return SwitchListTile(
          value: node.toggle.value,
          onChanged: (nextValue) {
            unawaited(
              _sendUiAction(
                componentId: componentId.isNotEmpty ? componentId : 'toggle',
                action: 'toggle',
                value: nextValue.toString(),
              ),
            );
          },
        );
      case uiv1.Node_Widget.dropdown:
        final componentId = _nodeId(node);
        final options = node.dropdown.options;
        final selected = options.contains(node.dropdown.value)
            ? node.dropdown.value
            : (options.isNotEmpty ? options.first : null);
        return DropdownButton<String>(
          isExpanded: true,
          value: selected,
          hint: const Text('Select option'),
          items: options
              .map(
                (option) => DropdownMenuItem<String>(
                  value: option,
                  child: Text(option),
                ),
              )
              .toList(),
          onChanged: options.isEmpty
              ? null
              : (nextValue) {
                  if (nextValue == null) {
                    return;
                  }
                  unawaited(
                    _sendUiAction(
                      componentId:
                          componentId.isNotEmpty ? componentId : 'dropdown',
                      action: 'select',
                      value: nextValue,
                    ),
                  );
                },
        );
      case uiv1.Node_Widget.image:
        return Image.network(
          node.image.url,
          fit: BoxFit.cover,
          errorBuilder: (context, error, stackTrace) {
            return const Icon(Icons.broken_image_outlined);
          },
        );
      case uiv1.Node_Widget.progress:
        return LinearProgressIndicator(
          value: node.progress.value.clamp(0.0, 1.0).toDouble(),
        );
      case uiv1.Node_Widget.notSet:
        break;
      default:
        return _renderNodeChildren(node.children);
    }
    return const SizedBox.shrink();
  }

  Widget _renderNodeChildren(List<uiv1.Node> children) {
    if (children.isEmpty) {
      return const SizedBox.shrink();
    }
    if (children.length == 1) {
      return _renderNode(children.first);
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: children.map(_renderNode).toList(),
    );
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
