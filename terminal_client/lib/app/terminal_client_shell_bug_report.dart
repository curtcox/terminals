part of 'terminal_client_shell.dart';

extension _BugReportExtension on _TerminalClientShellState {
  BugReceiptChromeState get _bugReceiptChromeState => _bugReceiptState;

  Future<void> _submitBugReport({
    required String subjectDeviceID,
    required String description,
    required diagv1.BugReportSource source,
    Map<String, String> sourceHints = const <String, String>{},
    List<String> tags = const <String>[],
    BugIdentifier? bugIdentifier,
  }) async {
    final now = DateTime.now().toLocal();
    final identifier = bugIdentifier ?? buildBugIdentifier(now);
    final screenshotPng = await _captureBugReportScreenshot();
    final bugReport = diagv1.BugReport()
      ..reportId = buildLocalBugReportId(
        now: now,
        identifier: identifier,
        reporterDeviceID: _deviceId,
        subjectDeviceID: subjectDeviceID,
      )
      ..reporterDeviceId = _deviceId
      ..subjectDeviceId = subjectDeviceID
      ..source = source
      ..description = description
      ..timestampUnixMs = Int64(now.toUtc().millisecondsSinceEpoch)
      ..clientContext = _buildClientContext();
    if (screenshotPng.isNotEmpty) {
      bugReport.screenshotPng = screenshotPng;
      bugReport.sourceHints['screenshot_byte_count'] =
          screenshotPng.length.toString();
    }
    bugReport.tags.addAll(<String>[
      ...tags,
      'bug_word:${identifier.word}',
      'bug_code:${identifier.code}',
    ]);
    bugReport.sourceHints['bug_token_word'] = identifier.word;
    bugReport.sourceHints['bug_token_code'] = identifier.code;
    bugReport.sourceHints['bug_token_timestamp_unix_ms'] =
        now.toUtc().millisecondsSinceEpoch.toString();
    _buildAutomaticBugSourceHints().forEach((key, value) {
      if (value.isEmpty) {
        return;
      }
      bugReport.sourceHints[key] = value;
    });
    sourceHints.forEach((key, value) {
      bugReport.sourceHints[key] = value;
    });

    if (!mounted) {
      return;
    }
    final firstQueuedUnixMs = _nowUnixMs();
    if (_isBugReportTransportReady()) {
      _dispatchBugReport(
        bugReport: bugReport,
        identifier: identifier,
        subjectDeviceID: subjectDeviceID,
        firstQueuedUnixMs: firstQueuedUnixMs,
        previousDispatchAttempts: 0,
        replay: false,
      );
      _rebuildState(() {
        _lastBugTokenWord = identifier.word;
        _lastBugTokenCode = identifier.code;
        _status = 'Bug report pending';
        _lastNotification =
            'Submitting bug report (word: ${identifier.word}, code: ${identifier.code}) and waiting for server ack...';
        _bugReceiptState = BugReceiptChromeState.waiting;
        _bugReceiptReportId = '';
        _bugReceiptDetail =
            'Waiting for server receipt for word ${identifier.word}, code ${identifier.code}.';
      });
      return;
    }

    _queuedBugReports.add(
      QueuedBugReport(
        bugReport: bugReport.deepCopy(),
        identifier: identifier,
        firstQueuedUnixMs: firstQueuedUnixMs,
        dispatchAttempts: 0,
      ),
    );
    final hasActiveStreamAttempt =
        _shouldStayConnected && _client != null && _incoming != null;
    if (!hasActiveStreamAttempt && !_isConnecting) {
      unawaited(_startStream(userInitiated: true));
    }
    _recordClientLog(
      'warn',
      'queued bug report while transport not ready for subject=$subjectDeviceID '
          'word=${identifier.word} code=${identifier.code} report_id=${bugReport.reportId}',
    );
    _rebuildState(() {
      _lastBugTokenWord = identifier.word;
      _lastBugTokenCode = identifier.code;
      _status = 'Bug report queued';
      _lastNotification =
          'Bug report queued (word: ${identifier.word}, code: ${identifier.code}). Connecting and will send automatically.';
      _bugReceiptState = BugReceiptChromeState.waiting;
      _bugReceiptReportId = '';
      _bugReceiptDetail =
          'Queued and waiting for a positive server receipt for word ${identifier.word}, code ${identifier.code}.';
    });
    _ensureBugReportAckWatchdog();
  }

  Future<void> _submitBugReportFromAction({
    required String componentId,
    required String action,
    required String value,
  }) async {
    var subjectDeviceID = _deviceId;
    final parts = action.split(':');
    if (parts.length > 1) {
      final explicit = parts.sublist(1).join(':').trim();
      if (explicit.isNotEmpty) {
        subjectDeviceID = explicit;
      }
    } else if (value.trim().isNotEmpty) {
      subjectDeviceID = value.trim();
    }

    await _submitBugReport(
      subjectDeviceID: subjectDeviceID,
      description: 'Filed from on-device bug report button',
      source: diagv1.BugReportSource.BUG_REPORT_SOURCE_SCREEN_BUTTON,
      sourceHints: <String, String>{
        'component_id': componentId,
        'action': action,
      },
    );
  }

  Future<void> _showBugReportDialog() async {
    final descriptionController = TextEditingController();
    final tagsController = TextEditingController();
    var draftIdentifier = buildBugIdentifier(DateTime.now().toLocal());
    final draft = await showDialog<BugReportDraft>(
      context: context,
      builder: (context) {
        return StatefulBuilder(
          builder: (context, setDialogState) {
            return AlertDialog(
              title: const Text('Report a bug'),
              content: SizedBox(
                width: 440,
                child: SingleChildScrollView(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('No fields are required.'),
                      const SizedBox(height: 12),
                      Container(
                        width: double.infinity,
                        padding: const EdgeInsets.all(12),
                        decoration: BoxDecoration(
                          border: Border.all(color: Colors.blueGrey.shade200),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            const Text(
                              'Reference word',
                              style: TextStyle(fontWeight: FontWeight.w600),
                            ),
                            const SizedBox(height: 6),
                            Text(
                              draftIdentifier.word,
                              style: Theme.of(context).textTheme.headlineSmall,
                            ),
                            SelectableText(draftIdentifier.code),
                            const SizedBox(height: 8),
                            Center(
                              child: QrImageView(
                                data: draftIdentifier.qrPayload,
                                size: 140,
                              ),
                            ),
                            const SizedBox(height: 8),
                            Wrap(
                              spacing: 8,
                              children: [
                                OutlinedButton.icon(
                                  onPressed: () =>
                                      _announceBugIdentifier(draftIdentifier),
                                  icon: const Icon(Icons.volume_up_outlined),
                                  label: const Text('Speak'),
                                ),
                                OutlinedButton.icon(
                                  onPressed: () {
                                    setDialogState(() {
                                      draftIdentifier = buildBugIdentifier(
                                        DateTime.now().toLocal(),
                                      );
                                    });
                                  },
                                  icon: const Icon(Icons.refresh_outlined),
                                  label: const Text('Refresh'),
                                ),
                              ],
                            ),
                          ],
                        ),
                      ),
                      const SizedBox(height: 12),
                      TextField(
                        controller: descriptionController,
                        minLines: 2,
                        maxLines: 4,
                        decoration: const InputDecoration(
                          labelText: 'Description (optional)',
                          hintText: 'What happened? (optional)',
                        ),
                      ),
                      const SizedBox(height: 12),
                      TextField(
                        controller: tagsController,
                        decoration: const InputDecoration(
                          labelText: 'Tags (optional)',
                          hintText: 'ui, playback, audio',
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              actions: [
                TextButton(
                  onPressed: () => Navigator.of(context).pop(),
                  child: const Text('Cancel'),
                ),
                FilledButton(
                  onPressed: () {
                    Navigator.of(context).pop(
                      BugReportDraft(
                        description: descriptionController.text.trim(),
                        tags: tagsController.text
                            .split(',')
                            .map((value) => value.trim())
                            .where((value) => value.isNotEmpty)
                            .toList(),
                        identifier: draftIdentifier,
                      ),
                    );
                  },
                  child: const Text('Submit'),
                ),
              ],
            );
          },
        );
      },
    );

    descriptionController.dispose();
    tagsController.dispose();
    if (draft == null) {
      return;
    }
    _announceBugIdentifier(draft.identifier);
    await _submitBugReport(
      subjectDeviceID: _deviceId,
      description: draft.description.isNotEmpty
          ? draft.description
          : 'Filed from terminal client bug-report dialog',
      source: diagv1.BugReportSource.BUG_REPORT_SOURCE_SCREEN_BUTTON,
      sourceHints: const <String, String>{
        'entry_point': 'manual_bug_report_dialog',
      },
      tags: draft.tags,
      bugIdentifier: draft.identifier,
    );
  }

  Future<List<int>> _captureBugReportScreenshot() async {
    try {
      final overrideCapture = widget.bugReportScreenshotCapture;
      if (overrideCapture != null) {
        return await overrideCapture();
      }
      final bindingType = WidgetsBinding.instance.runtimeType.toString();
      if (bindingType.contains('TestWidgetsFlutterBinding')) {
        return const <int>[];
      }
      if (!mounted) {
        return const <int>[];
      }
      final screenshotContext = _bugReportScreenshotKey.currentContext;
      if (screenshotContext == null) {
        return const <int>[];
      }
      final renderObject = screenshotContext.findRenderObject();
      if (renderObject is! RenderRepaintBoundary) {
        return const <int>[];
      }
      final pixelRatio = math.min(
        1.0,
        View.maybeOf(screenshotContext)?.devicePixelRatio ??
            _lastKnownDevicePixelRatio,
      );
      final image = await renderObject.toImage(
        pixelRatio: pixelRatio > 0 ? pixelRatio : 1.0,
      );
      try {
        final byteData = await image.toByteData(
          format: ui.ImageByteFormat.png,
        );
        if (byteData == null) {
          return const <int>[];
        }
        return Uint8List.sublistView(
          byteData.buffer.asUint8List(
            byteData.offsetInBytes,
            byteData.lengthInBytes,
          ),
        );
      } finally {
        image.dispose();
      }
    } catch (error) {
      _recordClientLog(
        'warn',
        'bug report screenshot capture failed error=$error',
      );
      return const <int>[];
    }
  }

  void _announceBugIdentifier(BugIdentifier identifier) {
    speech.speakText(
        'Bug reference word ${identifier.word}. Code ${identifier.code}');
    if (!mounted) {
      return;
    }
    final views = WidgetsBinding.instance.platformDispatcher.views;
    if (views.isEmpty) {
      return;
    }
    SemanticsService.sendAnnouncement(
      views.first,
      'Bug reference word ${identifier.word}. Code ${identifier.code}',
      _lastKnownTextDirection,
    );
  }

  Map<String, String> _buildAutomaticBugSourceHints() {
    return <String, String>{
      'host': _hostController.text.trim(),
      'port': _portController.text.trim(),
      'grpc_endpoint_override': _grpcEndpointController.text.trim(),
      'websocket_endpoint_override': _websocketEndpointController.text.trim(),
      'tcp_endpoint_override': _tcpEndpointController.text.trim(),
      'http_endpoint_override': _httpEndpointController.text.trim(),
      'status': _status,
      'last_connection_status': _lastConnectionStatus,
      'active_ui_root':
          _activeRoot == null ? '' : serverDrivenNodeId(_activeRoot!),
      'active_stream_count': _activeStreamsByID.length.toString(),
      'route_count': _routesByStreamID.length.toString(),
      'reconnect_attempt': _reconnectAttempt.toString(),
      'last_notification': _lastNotification,
      'last_transport_diagnostic': _lastTransportDiagnostic,
      'page_url': Uri.base.toString(),
      'active_root_revision': _activeRootRevision.toString(),
      'responses_seen': _responses.toString(),
      'queued_bug_reports': _queuedBugReports.length.toString(),
      'pending_bug_reports': _pendingBugReports.length.toString(),
    };
  }

  void _handleBugReportAck(diagv1.BugReportAck ack) {
    PendingBugReport? pending;
    if (_pendingBugReports.isNotEmpty) {
      pending = _pendingBugReports.removeAt(0);
    }
    final tokenWord = pending?.identifier.word ?? _lastBugTokenWord;
    final tokenCode = pending?.identifier.code ?? _lastBugTokenCode;
    _lastBugTokenWord = tokenWord;
    _lastBugTokenCode = tokenCode;
    if (_pendingBugReports.isEmpty) {
      _bugReportAckTimer?.cancel();
      _bugReportAckTimer = null;
    }
    final receiptID = ack.reportId.trim();
    if (receiptID.isEmpty) {
      _status = 'Bug report receipt error';
      _lastNotification = tokenWord.isEmpty
          ? 'Bug report failed: no positive receipt from server.'
          : 'Bug report failed: no positive receipt from server (word: $tokenWord).';
      _bugReceiptState = BugReceiptChromeState.error;
      _bugReceiptReportId = '';
      _bugReceiptDetail = tokenWord.isEmpty
          ? 'No positive receipt was generated by the server.'
          : 'No positive receipt was generated by the server for word $tokenWord, code $tokenCode.';
      _recordClientLog(
        'error',
        'bug report ack missing report_id status=${ack.status.name} '
            'word=$tokenWord code=$tokenCode',
      );
      return;
    }
    _status = 'Bug report filed';
    _lastNotification = tokenWord.isEmpty
        ? 'Bug report filed: $receiptID'
        : 'Bug report filed: $receiptID (word: $tokenWord)';
    _bugReceiptState = BugReceiptChromeState.received;
    _bugReceiptReportId = receiptID;
    final ackMessage = ack.message.trim();
    if (ackMessage == 'ack_replayed') {
      _bugReceiptDetail = tokenWord.isEmpty
          ? 'Positive receipt recovered after transport failover. The server replayed the original acknowledgement.'
          : 'Positive receipt recovered after transport failover for word $tokenWord, code $tokenCode.';
    } else {
      _bugReceiptDetail = tokenWord.isEmpty
          ? 'Positive receipt received from server.'
          : 'Positive receipt received for word $tokenWord, code $tokenCode.';
    }
    _recordClientLog(
      'info',
      'bug report ack status=${ack.status.name} id=$receiptID '
          'word=$tokenWord code=$tokenCode message=${ack.message}',
    );
  }

  void _failPendingBugReportsForControlError(ControlError error) {
    if (_pendingBugReports.isEmpty) {
      return;
    }
    final failed = _pendingBugReports.removeAt(0);
    _lastBugTokenWord = failed.identifier.word;
    _lastBugTokenCode = failed.identifier.code;
    final reason = error.message.trim().isNotEmpty
        ? error.message.trim()
        : error.code.name;
    _status = 'Bug report receipt error';
    _lastNotification =
        'Bug report failed: $reason (word: ${failed.identifier.word}, code: ${failed.identifier.code}).';
    _bugReceiptState = BugReceiptChromeState.error;
    _bugReceiptReportId = '';
    _bugReceiptDetail =
        'No positive receipt could be generated: $reason.${_bugTransportContextSuffix()}';
    _recordClientLog(
      'error',
      'bug report rejected by control error code=${error.code.name} '
          'message=$reason word=${failed.identifier.word} code=${failed.identifier.code}',
    );
    _pendingBugReports.clear();
    _bugReportAckTimer?.cancel();
    _bugReportAckTimer = null;
  }

  bool _isBugReportTransportReady() {
    return _isConnectionRegistered;
  }

  void _dispatchBugReport({
    required diagv1.BugReport bugReport,
    required BugIdentifier identifier,
    required String subjectDeviceID,
    required int firstQueuedUnixMs,
    required int previousDispatchAttempts,
    required bool replay,
  }) {
    unawaited(
      _sendWhenReady(
        operation: OutboundOperation.bugReport,
        request: ConnectRequest()..bugReport = bugReport.deepCopy(),
      ),
    );
    final nextAttempt = previousDispatchAttempts + 1;
    _pendingBugReports.add(
      PendingBugReport(
        bugReport: bugReport.deepCopy(),
        identifier: identifier,
        firstQueuedUnixMs: firstQueuedUnixMs,
        submittedUnixMs: _nowUnixMs(),
        dispatchAttempts: nextAttempt,
      ),
    );
    _ensureBugReportAckWatchdog();
    _recordClientLog(
      'info',
      '${replay ? 'replayed' : 'submitted'} bug report for subject=$subjectDeviceID '
          'word=${identifier.word} code=${identifier.code} '
          'report_id=${bugReport.reportId} attempt=$nextAttempt',
    );
  }

  void _flushQueuedBugReports() {
    if (!_isBugReportTransportReady()) {
      return;
    }
    if (_queuedBugReports.isEmpty) {
      return;
    }
    final queued = List<QueuedBugReport>.from(_queuedBugReports);
    _queuedBugReports.clear();
    for (final item in queued) {
      _dispatchBugReport(
        bugReport: item.bugReport.deepCopy(),
        identifier: item.identifier,
        subjectDeviceID: item.bugReport.subjectDeviceId,
        firstQueuedUnixMs: item.firstQueuedUnixMs,
        previousDispatchAttempts: item.dispatchAttempts,
        replay: true,
      );
    }
    if (mounted) {
      _rebuildState(() {
        _status = 'Queued bug reports sent';
        _lastNotification =
            'Sent ${queued.length} queued bug report(s); waiting for server ack.';
        _bugReceiptState = BugReceiptChromeState.waiting;
        _bugReceiptReportId = '';
        _bugReceiptDetail = queued.length == 1
            ? 'Queued bug report sent. Waiting for positive server receipt.'
            : 'Sent ${queued.length} queued bug reports. Waiting for positive server receipts.';
      });
    }
  }

  void _ensureBugReportAckWatchdog() {
    if (_bugReportAckTimer != null) {
      return;
    }
    _bugReportAckTimer = Timer.periodic(_bugReportAckRetryPolicy.interval, (_) {
      final nowUnixMs = _nowUnixMs();
      if (_pendingBugReports.isNotEmpty) {
        final first = _pendingBugReports.first;
        if (_bugReportAckRetryPolicy.hasTimedOut(
          Duration(milliseconds: nowUnixMs - first.firstQueuedUnixMs),
        )) {
          final failed = _pendingBugReports.removeAt(0);
          _lastBugTokenWord = failed.identifier.word;
          _lastBugTokenCode = failed.identifier.code;
          if (mounted) {
            _rebuildState(() {
              _status = 'Bug report receipt error';
              _lastNotification =
                  'Bug report failed: no positive receipt from server (word: ${failed.identifier.word}, code: ${failed.identifier.code}).';
              _bugReceiptState = BugReceiptChromeState.error;
              _bugReceiptReportId = '';
              _bugReceiptDetail =
                  'No positive receipt was generated by the server for word ${failed.identifier.word}, code ${failed.identifier.code}.${_bugTransportContextSuffix()}';
            });
          }
          _recordClientLog(
            'error',
            'bug report ack timeout word=${failed.identifier.word} code=${failed.identifier.code} '
                'attempts=${failed.dispatchAttempts} transport_diagnostic=$_lastTransportDiagnostic',
          );
        }
      } else if (_queuedBugReports.isNotEmpty &&
          _bugReceiptState == BugReceiptChromeState.waiting) {
        final firstQueued = _queuedBugReports.first;
        if (_bugReportAckRetryPolicy.hasTimedOut(
          Duration(milliseconds: nowUnixMs - firstQueued.firstQueuedUnixMs),
        )) {
          _lastBugTokenWord = firstQueued.identifier.word;
          _lastBugTokenCode = firstQueued.identifier.code;
          if (mounted) {
            _rebuildState(() {
              _status = 'Bug report receipt error';
              _lastNotification =
                  'Bug report failed: no positive receipt while queued (word: ${firstQueued.identifier.word}, code: ${firstQueued.identifier.code}).';
              _bugReceiptState = BugReceiptChromeState.error;
              _bugReceiptReportId = '';
              _bugReceiptDetail =
                  'No positive receipt could be generated because the report remained queued for more than ${_bugReportAckRetryPolicy.maxDuration.inSeconds}s.${_bugTransportContextSuffix()}';
            });
          }
          _recordClientLog(
            'error',
            'queued bug report receipt timeout word=${firstQueued.identifier.word} code=${firstQueued.identifier.code} '
                'transport_diagnostic=$_lastTransportDiagnostic',
          );
        }
      }
      if (_pendingBugReports.isEmpty &&
          (_queuedBugReports.isEmpty ||
              _bugReceiptState != BugReceiptChromeState.waiting)) {
        _bugReportAckTimer?.cancel();
        _bugReportAckTimer = null;
      }
    });
  }

  void _drainPendingBugReportsAsFailed(String reason) {
    if (_pendingBugReports.isEmpty) {
      return;
    }
    final failed = _pendingBugReports.removeAt(0);
    _lastBugTokenWord = failed.identifier.word;
    _lastBugTokenCode = failed.identifier.code;
    _recordClientLog(
      'error',
      'bug report failed word=${failed.identifier.word} code=${failed.identifier.code} reason=$reason',
    );
    if (mounted) {
      _rebuildState(() {
        _status = 'Bug report receipt error';
        _lastNotification =
            'Bug report failed: no positive receipt (word: ${failed.identifier.word}, code: ${failed.identifier.code}).';
        _bugReceiptState = BugReceiptChromeState.error;
        _bugReceiptReportId = '';
        _bugReceiptDetail = 'No positive receipt could be generated: $reason.';
      });
    }
    _pendingBugReports.clear();
    _bugReportAckTimer?.cancel();
    _bugReportAckTimer = null;
  }

  void _requeuePendingBugReportsForRetry(String reason) {
    if (_pendingBugReports.isEmpty) {
      return;
    }
    final pending = List<PendingBugReport>.from(_pendingBugReports);
    _pendingBugReports.clear();
    _queuedBugReports.insertAll(
      0,
      pending
          .map(
            (item) => QueuedBugReport(
              bugReport: item.bugReport.deepCopy(),
              identifier: item.identifier,
              firstQueuedUnixMs: item.firstQueuedUnixMs,
              dispatchAttempts: item.dispatchAttempts,
            ),
          )
          .toList(),
    );
    final first = pending.first;
    _lastBugTokenWord = first.identifier.word;
    _lastBugTokenCode = first.identifier.code;
    _recordClientLog(
      'warn',
      're-queued ${pending.length} pending bug report(s) after transport failure: $reason',
    );
    if (!mounted) {
      return;
    }
    _rebuildState(() {
      _status = 'Bug report retry pending';
      _lastNotification =
          'Retrying bug report after transport recovery (word: ${first.identifier.word}, code: ${first.identifier.code}).';
      _bugReceiptState = BugReceiptChromeState.waiting;
      _bugReceiptReportId = '';
      _bugReceiptDetail =
          'Still waiting for a positive server receipt for word ${first.identifier.word}, code ${first.identifier.code}. Last failure: $reason';
    });
  }

  String _bugTransportContextSuffix() {
    final diagnostic = _lastTransportDiagnostic.trim();
    if (diagnostic.isEmpty) {
      return '';
    }
    return ' Last transport diagnostic: $diagnostic';
  }
}
