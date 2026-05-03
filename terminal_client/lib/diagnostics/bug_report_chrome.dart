import 'package:flutter/material.dart';
import 'package:terminal_client/gen/terminals/diagnostics/v1/diagnostics.pb.dart'
    as diagv1;

const String bugReportActionPrefix = 'bug_report';

const List<String> bugTokenWords = <String>[
  'ace',
  'actor',
  'adapt',
  'air',
  'alert',
  'anchor',
  'angle',
  'apple',
  'artist',
  'asset',
  'audio',
  'autumn',
  'badge',
  'balance',
  'beam',
  'berry',
  'beyond',
  'bicycle',
  'bird',
  'blossom',
  'blue',
  'board',
  'book',
  'breeze',
  'bridge',
  'bright',
  'buffer',
  'button',
  'cable',
  'calm',
  'camera',
  'canvas',
  'captain',
  'carbon',
  'care',
  'center',
  'chapter',
  'check',
  'choice',
  'circle',
  'city',
  'clarity',
  'clock',
  'cloud',
  'coast',
  'color',
  'comfort',
  'compass',
  'control',
  'copper',
  'corner',
  'cotton',
  'craft',
  'credit',
  'crisp',
  'current',
  'cycle',
  'daily',
  'dawn',
  'delta',
  'design',
  'detail',
  'device',
  'dialog',
  'dock',
  'domain',
  'dream',
  'drift',
  'drive',
  'echo',
  'edge',
  'ember',
  'energy',
  'engine',
  'entry',
  'equal',
  'estate',
  'evening',
  'event',
  'fabric',
  'factor',
  'field',
  'filter',
  'final',
  'flame',
  'flight',
  'flower',
  'focus',
  'forest',
  'frame',
  'fresh',
  'future',
  'garden',
  'gentle',
  'glide',
  'glow',
  'gold',
  'grain',
  'graph',
  'green',
  'group',
  'guide',
  'habit',
  'harbor',
  'harmony',
  'haven',
  'hero',
  'horizon',
  'house',
  'idea',
  'image',
  'index',
  'island',
  'item',
  'jewel',
  'journey',
  'keeper',
  'key',
  'kind',
  'kit',
  'ladder',
  'lake',
  'launch',
  'layer',
  'leaf',
  'legend',
  'level',
  'light',
  'limit',
  'linen',
  'list',
  'logic',
  'lucky',
  'lumen',
  'maker',
  'map',
  'market',
  'matrix',
  'meadow',
  'memory',
  'metal',
  'method',
  'metric',
  'midday',
  'mind',
  'mirror',
  'model',
  'moment',
  'monsoon',
  'morning',
  'motion',
  'mountain',
  'music',
  'native',
  'nature',
  'network',
  'nexus',
  'night',
  'noble',
  'north',
  'note',
  'novel',
  'number',
  'oak',
  'object',
  'ocean',
  'offer',
  'omega',
  'onward',
  'orbit',
  'origin',
  'output',
  'packet',
  'page',
  'panel',
  'paper',
  'path',
  'pearl',
  'pencil',
  'pepper',
  'photo',
  'pixel',
  'planet',
  'plate',
  'point',
  'portal',
  'power',
  'prairie',
  'prime',
  'pulse',
  'quiet',
  'rapid',
  'reader',
  'record',
  'reef',
  'render',
  'report',
  'ribbon',
  'river',
  'rocket',
  'root',
  'round',
  'route',
  'sail',
  'sample',
  'scale',
  'scene',
  'screen',
  'script',
  'sea',
  'seed',
  'shadow',
  'signal',
  'silver',
  'simple',
  'sky',
  'smile',
  'snow',
  'solar',
  'source',
  'spark',
  'spirit',
  'spring',
  'square',
  'stable',
  'stage',
  'star',
  'stone',
  'storm',
  'story',
  'stream',
  'street',
  'studio',
  'summer',
  'sun',
  'sunset',
  'switch',
  'table',
  'target',
  'task',
  'tempo',
  'text',
  'thread',
  'timber',
  'today',
  'token',
  'tower',
  'trace',
  'track',
  'travel',
  'tree',
  'trust',
  'tunnel',
  'union',
  'unit',
  'update',
  'urban',
  'value',
  'vector',
  'velvet',
  'view',
  'vivid',
  'voice',
  'wave',
  'weather',
  'window',
  'winter',
  'wisdom',
  'wood',
  'world',
  'writer',
  'yard',
  'year',
  'yield',
  'young',
  'zenith',
  'zone',
];

enum BugReceiptChromeState {
  none,
  waiting,
  received,
  error,
}

class BugReportButton extends StatelessWidget {
  const BugReportButton({
    super.key,
    required this.onPressed,
  });

  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return FloatingActionButton.extended(
      onPressed: onPressed,
      icon: const Icon(Icons.bug_report_outlined),
      label: const Text('Report Bug'),
    );
  }
}

class BugIdentifier {
  const BugIdentifier({
    required this.word,
    required this.code,
    required this.qrPayload,
  });

  final String word;
  final String code;
  final String qrPayload;
}

class BugReportDraft {
  const BugReportDraft({
    required this.description,
    required this.tags,
    required this.identifier,
  });

  final String description;
  final List<String> tags;
  final BugIdentifier identifier;
}

class PendingBugReport {
  const PendingBugReport({
    required this.bugReport,
    required this.identifier,
    required this.firstQueuedUnixMs,
    required this.submittedUnixMs,
    required this.dispatchAttempts,
  });

  final diagv1.BugReport bugReport;
  final BugIdentifier identifier;
  final int firstQueuedUnixMs;
  final int submittedUnixMs;
  final int dispatchAttempts;
}

class QueuedBugReport {
  const QueuedBugReport({
    required this.bugReport,
    required this.identifier,
    required this.firstQueuedUnixMs,
    required this.dispatchAttempts,
  });

  final diagv1.BugReport bugReport;
  final BugIdentifier identifier;
  final int firstQueuedUnixMs;
  final int dispatchAttempts;
}

BugIdentifier buildBugIdentifier(
  DateTime nowLocal, {
  List<String> words = bugTokenWords,
}) {
  final tokenWords = words.isEmpty ? bugTokenWords : words;
  final secondsFromMidnight =
      nowLocal.hour * 3600 + nowLocal.minute * 60 + nowLocal.second;
  final daySalt = nowLocal.year * 10000 + nowLocal.month * 100 + nowLocal.day;
  final index = ((secondsFromMidnight ~/ 17) + daySalt) % tokenWords.length;
  final word = tokenWords[index];
  final hh = nowLocal.hour.toString().padLeft(2, '0');
  final mm = nowLocal.minute.toString().padLeft(2, '0');
  final ss = nowLocal.second.toString().padLeft(2, '0');
  final code = '$hh$mm$ss-$word';
  return BugIdentifier(
    word: word,
    code: code,
    qrPayload: 'terminals-bug://$code',
  );
}

String sanitizeBugReportIdComponent(String value) {
  final normalized = value
      .trim()
      .toLowerCase()
      .replaceAll(RegExp(r'[^a-z0-9]+'), '-')
      .replaceAll(RegExp(r'^-+'), '')
      .replaceAll(RegExp(r'-+$'), '');
  if (normalized.isEmpty) {
    return 'unknown';
  }
  return normalized;
}

String buildLocalBugReportId({
  required DateTime now,
  required BugIdentifier identifier,
  required String reporterDeviceID,
  required String subjectDeviceID,
}) {
  final reporter = sanitizeBugReportIdComponent(reporterDeviceID);
  final subject = sanitizeBugReportIdComponent(subjectDeviceID);
  final code = sanitizeBugReportIdComponent(identifier.code);
  return 'clientbug-${now.toUtc().millisecondsSinceEpoch}-$reporter-$subject-$code';
}

class BugReceiptPanel extends StatelessWidget {
  const BugReceiptPanel({
    super.key,
    required this.state,
    this.reportId = '',
    this.detail = '',
  });

  final BugReceiptChromeState state;
  final String reportId;
  final String detail;

  @override
  Widget build(BuildContext context) {
    late final Color borderColor;
    late final Color backgroundColor;
    late final IconData icon;
    late final String title;
    switch (state) {
      case BugReceiptChromeState.none:
        return const SizedBox.shrink();
      case BugReceiptChromeState.waiting:
        borderColor = Colors.amber.shade400;
        backgroundColor = Colors.amber.shade50;
        icon = Icons.schedule_outlined;
        title = 'Bug Report Receipt: Pending';
        break;
      case BugReceiptChromeState.received:
        borderColor = Colors.green.shade400;
        backgroundColor = Colors.green.shade50;
        icon = Icons.verified_outlined;
        title = 'Bug Report Receipt: Received';
        break;
      case BugReceiptChromeState.error:
        borderColor = Colors.red.shade400;
        backgroundColor = Colors.red.shade50;
        icon = Icons.error_outline;
        title = 'Bug Report Receipt: Error';
        break;
    }
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        border: Border.all(color: borderColor),
        borderRadius: BorderRadius.circular(8),
        color: backgroundColor,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 18),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: const TextStyle(fontWeight: FontWeight.w600),
                ),
                if (reportId.isNotEmpty)
                  Text(
                    'Receipt ID: $reportId',
                    style: const TextStyle(fontSize: 12),
                  ),
                if (detail.isNotEmpty)
                  Text(detail, style: const TextStyle(fontSize: 12)),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
