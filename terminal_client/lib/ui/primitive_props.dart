import 'package:flutter/material.dart';

Color? parseHexColor(String? raw) {
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
