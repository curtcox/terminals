/// Runtime policy flags applied to edge operator execution.
class SandboxPolicy {
  const SandboxPolicy({
    this.allowNetwork = false,
    this.allowSubprocess = false,
    this.allowFilesystemWrites = false,
  });

  final bool allowNetwork;
  final bool allowSubprocess;
  final bool allowFilesystemWrites;
}
