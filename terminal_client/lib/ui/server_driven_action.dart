class ServerDrivenAction {
  const ServerDrivenAction({
    required this.componentId,
    required this.action,
    this.value = '',
  });

  final String componentId;
  final String action;
  final String value;
}
