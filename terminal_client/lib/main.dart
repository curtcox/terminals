import 'package:flutter/material.dart';

void main() {
  runApp(const TerminalClientApp());
}

class TerminalClientApp extends StatelessWidget {
  const TerminalClientApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Terminal Client',
      home: const Scaffold(
        body: Center(
          child: Text('Terminal client scaffold'),
        ),
      ),
    );
  }
}

