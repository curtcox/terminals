# discovery

mDNS-based server discovery.

`mdns_scanner.dart` scans the local network for Terminals servers advertising via multicast DNS and emits `DiscoveredServer` records. Used by the connection screen to populate the server list without manual IP entry.
