import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;

String serverDrivenNodeId(uiv1.Node node) {
  if (node.id.isNotEmpty) {
    return node.id;
  }
  return node.props['id'] ?? '';
}
