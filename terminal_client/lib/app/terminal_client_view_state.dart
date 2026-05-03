import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;

enum ClientChromeMode {
  standard,
  hidden,
}

ClientChromeMode clientChromeModeFromRoot(uiv1.Node? root) {
  if (root == null) {
    return ClientChromeMode.standard;
  }
  final rawMode =
      (root.props['client_chrome'] ?? root.props['chrome'] ?? '').trim();
  switch (rawMode.toLowerCase()) {
    case 'hidden':
    case 'hide':
    case 'fullscreen':
    case 'immersive':
      return ClientChromeMode.hidden;
    default:
      return ClientChromeMode.standard;
  }
}

bool shouldHideClientChromeForRoot(uiv1.Node? root) {
  return clientChromeModeFromRoot(root) == ClientChromeMode.hidden;
}
