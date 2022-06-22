# NeoGo node configuration file

This section contains detailed NeoGo node configuration file description
including default config values and some tips to set up configurable values.

Each config file contains two sections. `ApplicationConfiguration` describes node-related
settings and `ProtocolConfiguration` contains protocol-related settings. See the
[Application Configuration](#Application-Configuration) and
[Protocol Configuration](#Protocol-Configuration) sections for details on configurable
values.

## Application Configuration

`ApplicationConfiguration` section of `yaml` node configuration file contains
node-related settings described in the table below.

| Section | Type | Default value | Description |
| --- | --- | --- | --- |
| Address | `string` | `127.0.0.1` | Node address that P2P protocol handler binds to. |
| AnnouncedPort | `uint16` | Same as `NodePort` | Node port which should be used to announce node's port on P2P layer, it can differ from the `NodePort` the node is bound to (for example, if your node is behind NAT). |
| AttemptConnPeers | `int` | `20` | Number of connection to try to establish when the connection count drops below the `MinPeers` value.|
| DBConfiguration | [DB Configuration](#DB-Configuration) |  | Describes configuration for database. See the [DB Configuration](#DB-Configuration) section for details. |
| DialTimeout | `int64` | `0` | Maximum duration a single dial may take in seconds. |
| ExtensiblePoolSize | `int` | `20` | Maximum amount of the extensible payloads from a single sender stored in a local pool. |
| LogPath | `string` | "", so only console logging | File path where to store node logs. |
| MaxPeers | `int` | `100` | Maximum numbers of peers that can be connected to the server. |
| MinPeers | `int` | `5` | Minimum number of peers for normal operation; when the node has less than this number of peers it tries to connect with some new ones. |
| NodePort | `uint16` | `0`, which is any free port | The actual node port it is bound to. |
| Oracle | [Oracle Configuration](#Oracle-Configuration) | | Oracle module configuration. See the [Oracle Configuration](#Oracle-Configuration) section for details. |
| P2PNotary | [P2P Notary Configuration](#P2P-Notary-Configuration) | | P2P Notary module configuration. See the [P2P Notary Configuration](#P2P-Notary-Configuration) section for details. |
| PingInterval | `int64` | `30` | Interval in seconds used in pinging mechanism for syncing blocks. |
| PingTimeout | `int64` | `90` | Time to wait for pong (response for sent ping request). |
| Pprof | [Metrics Services Configuration](#Metrics-Services-Configuration) | | Configuration for pprof service (profiling statistics gathering). See the [Metrics Services Configuration](#Metrics-Services-Configuration) section for details. |
| Prometheus | [Metrics Services Configuration](#Metrics-Services-Configuration) | | Configuration for Prometheus (monitoring system). See the [Metrics Services Configuration](#Metrics-Services-Configuration) section for details |
| ProtoTickInterval | `int64` | `5` | Duration in seconds between protocol ticks with each connected peer. |
| Relay | `bool` | `true` | Determines whether the server is forwarding its inventory. |
| RPC | [RPC Configuration](#RPC-Configuration) |  | Describes [RPC subsystem](rpc.md) configuration. See the [RPC Configuration](#RPC-Configuration) for details. |
| StateRoot | [State Root Configuration](#State-Root-Configuration) |  | State root module configuration. See the [State Root Configuration](#State-Root-Configuration) section for details. |
| UnlockWallet | [Unlock Wallet Configuration](#Unlock-Wallet-Configuration) |  | Node wallet configuration used for consensus (dBFT) operation. See the [Unlock Wallet Configuration](#Unlock-Wallet-Configuration) section for details. |

### DB Configuration

`DBConfiguration` section describes configuration for node database and has
the following format:
```
DBConfiguration:
  Type: leveldb
  LevelDBOptions:
    DataDirectoryPath: /chains/privnet
  BoltDBOptions:
    FilePath: ./chains/privnet.bolt
```
where:
- `Type` is the database type (string value). Supported types: `levelDB` and
  `boltDB`.
- `LevelDBOptions` are settings for LevelDB.
- `BoltDBOptions` configures BoltDB.

Only options for the specified database type will be used.

### Oracle Configuration

`Oracle` configuration section describes configuration for Oracle node module
and has the following structure:
```
Oracle:
  Enabled: false
  AllowPrivateHost: false
  MaxTaskTimeout: 3600s
  MaxConcurrentRequests: 10
  Nodes: ["172.200.0.1:30333", "172.200.0.2:30334"]
  NeoFS:
    Nodes: ["172.200.0.1:30335", "172.200.0.2:30336"]
    Timeout: 2
  RefreshInterval: 180s
  RequestTimeout: 5s
  ResponseTimeout: 5s
  UnlockWallet:
    Path: "./oracle_wallet.json"
    Password: "pass"
```

Please, refer to the [Oracle module documentation](./oracle.md#Configuration) for
details on configurable values.

### P2P Notary Configuration

`P2PNotary` configuration section describes configuration for P2P Notary node
module and has the following structure:
```
P2PNotary:
  Enabled: false
  UnlockWallet:
    Path: "/notary_wallet.json"
    Password: "pass"
```
where:
- `Enabled` denotes whether P2P Notary module is active.
- `UnlockWallet` is a Notary node wallet configuration, see the
  [Unlock Wallet Configuration](#Unlock-Wallet-Configuration) section for
  structure details.

Please, refer to the [Notary module documentation](./notary.md#Notary node module) for
details on module features.

### Metrics Services Configuration

Metrics services configuration describes options for metrics services (pprof,
Prometheus) and has the following structure:
```
Pprof:
  Enabled: false
  Address: ""
  Port: "30001"
Prometheus:
  Enabled: false
  Address: ""
  Port: "40001"
```
where:
- `Enabled` denotes whether the service is enabled.
- `Address` is a service address to be running at.
- `Port` is a service port to be bound to.

### RPC Configuration

`RPC` configuration section describes settings for the RPC server and has
the following structure:
```
RPC:
  Enabled: true
  Address: ""
  EnableCORSWorkaround: false
  MaxGasInvoke: 50
  MaxIteratorResultItems: 100
  MaxFindResultItems: 100
  MaxNEP11Tokens: 100
  Port: 10332
  SessionEnabled: true
  SessionExpirationTime: 60
  StartWhenSynchronized: false
  TLSConfig:
    Address: ""
    CertFile: serv.crt
    Enabled: true
    Port: 10331
    KeyFile: serv.key
```
where:
- `Enabled` denotes whether an RPC server should be started.
- `Address` is an RPC server address to be running at.
- `EnableCORSWorkaround` enables Cross-Origin Resource Sharing and is useful if
  you're accessing RPC interface from the browser.
- `MaxGasInvoke` is the maximum GAS allowed to spend during `invokefunction` and
  `invokescript` RPC-calls.
- `MaxIteratorResultItems` - maximum number of elements extracted from iterator
   returned by `invoke*` call. When the `MaxIteratorResultItems` value is set to
   `n`, only `n` iterations are returned and truncated is true, indicating that
   there is still data to be returned.
- `MaxFindResultItems` - the maximum number of elements for `findstates` response.
- `MaxNEP11Tokens` - limit for the number of tokens returned from
  `getnep11balances` call.
- `Port` is an RPC server port it should be bound to.
- `SessionEnabled` denotes whether session-based iterator JSON-RPC API is enabled.
  If true, then all iterators got from `invoke*` calls will be stored as sessions
  on the server side available for further traverse. `traverseiterator` and
  `terminatesession` JSON-RPC calls will be handled by the server. It is not
  recommended to enable this setting on public networks due to possible DoS
  attack. Set to `false` by default.
- `SessionExpirationTime` is a lifetime of iterator session in seconds. It is set
  to `60` seconds by default and is relevant only if `SessionEnabled` is set to
  `true`.
- `StartWhenSynchronized` controls when RPC server will be started, by default
  (`false` setting) it's started immediately and RPC is availabe during node
  synchronization. Setting it to `true` will make the node start RPC service only
  after full synchronization.
- `TLS` section configures TLS protocol.

### State Root Configuration

`StateRoot` configuration section contains settings for state roots exchange and has
the following structure:
```
StateRoot:
  Enabled: false
  UnlockWallet:
    Path: "./wallet.json"
    Password: "pass"
```
where:
- `Enabled` enables state root module.
- `UnlockWallet` contains wallet settings, see
  [Unlock Wallet Configuration](#Unlock-Wallet-Configuration) section for
  structure details.

### Unlock Wallet Configuration

`UnlockWallet` configuration section contains wallet settings and has the following
structure:
```
UnlockWallet:
  Path: "./wallet.json"
  Password: "pass"
```
where:
- `Path` is a path to wallet.
- `Password` is a wallet password.

## Protocol Configuration

`ProtocolConfiguration` section of `yaml` node configuration file contains
protocol-related settings described in the table below.

| Section | Type | Default value | Description | Notes |
| --- | --- | --- | --- | --- |
| CommitteeHistory | map[uint32]int | none | Number of committee members after the given height, for example `{0: 1, 20: 4}` sets up a chain with one committee member since the genesis and then changes the setting to 4 committee members at the height of 20. `StandbyCommittee` committee setting must have the number of keys equal or exceeding the highest value in this option. Blocks numbers where the change happens must be divisible by the old and by the new values simultaneously. If not set, committee size is derived from the `StandbyCommittee` setting and never changes. |
| GarbageCollectionPeriod | `uint32` | 10000 | Controls MPT garbage collection interval (in blocks) for configurations with `RemoveUntraceableBlocks` enabled and `KeepOnlyLatestState` disabled. In this mode the node stores a number of MPT trees (corresponding to `MaxTraceableBlocks` and `StateSyncInterval`), but the DB needs to be clean from old entries from time to time. Doing it too often will cause too much processing overhead, doing it too rarely will leave more useless data in the DB. |
| Hardforks | `map[string]uint32` | [] | The set of incompatible changes that affect node behaviour starting from the specified height. The default value is an empty set which should be interpreted as "each known hard-fork is applied from the zero blockchain height". The list of valid hard-fork names:<br>• `Aspidochelone` represents hard-fork introduced in [#2469](https://github.com/nspcc-dev/neo-go/pull/2469) (ported from the [reference](https://github.com/neo-project/neo/pull/2712)). It adjusts the prices of `System.Contract.CreateStandardAccount` and `System.Contract.CreateMultisigAccount` interops so that the resulting prices are in accordance with `sha256` method of native `CryptoLib` contract. `Aspidochelone` is also includes [#2519](https://github.com/nspcc-dev/neo-go/pull/2519) (ported from the [reference](https://github.com/neo-project/neo/pull/2749)). It adjusts the price of `System.Runtime.GetRandom` interop and fixes its vulnerability. |
| KeepOnlyLatestState | `bool` | `false` | Specifies if MPT should only store latest state. If true, DB size will be smaller, but older roots won't be accessible. This value should remain th
e same for the same database. | Conflicts with `P2PStateExchangeExtensions`. |
| Magic | `uint32` | `0` | Magic number which uniquely identifies NEO network. |
| MaxBlockSize | `uint32` | `262144` | Maximum block size in bytes. |
| MaxBlockSystemFee | `int64` | `900000000000` | Maximum overall transactions system fee per block. |
| MaxTraceableBlocks | `uint32` | `2102400` | Length of the chain accessible to smart contracts. | `RemoveUntraceableBlocks` should be enabled to use this setting. |
| MaxTransactionsPerBlock | `uint16` | `512` | Maximum number of transactions per block. |
| MemPoolSize | `int` | `50000` | Size of the node's memory pool where transactions are stored before they are added to block. |
| NativeActivations | `map[string][]uint32` | ContractManagement: [0]<br>StdLib: [0]<br>CryptoLib: [0]<br>LedgerContract: [0]<br>NeoToken: [0]<br>GasToken: [0]<br>PolicyContract: [0]<br>RoleManagement: [0]<br>OracleContract: [0] | The list of histories of native contracts updates. Each list item shod be presented as a known native contract name with the corresponding list of chain's heights. The contract is not active until chain reaches the first height value specified in the list. | `Notary` is supported. |
| P2PNotaryRequestPayloadPoolSize | `int` | `1000` | Size of the node's P2P Notary request payloads memory pool where P2P Notary requests are stored before main or fallback transaction is completed and added to the chain.<br>This option is valid only if `P2PSigExtensions` are enabled. | Not supported by the C# node, thus may affect heterogeneous networks functionality. |
| P2PSigExtensions | `bool` | `false` | Enables following additional Notary service related logic:<br>• Transaction attributes `NotValidBefore`, `Conflicts` and `NotaryAssisted`<br>• Network payload of the `P2PNotaryRequest` type<br>• Native `Notary` contract<br>• Notary node module | Not supported by the C# node, thus may affect heterogeneous networks functionality. |
| P2PStateExchangeExtensions | `bool` | `false` | Enables following P2P MPT state data exchange logic: <br>• `StateSyncInterval` protocol setting <br>• P2P commands `GetMPTDataCMD` and `MPTDataCMD` | Not supported by the C# node, thus may affect heterogeneous networks functionality. Conflicts with `KeepOnlyLatestState`. |
| RemoveUntraceableBlocks | `bool`| `false` | Denotes whether old blocks should be removed from cache and database. If enabled, then only last `MaxTraceableBlocks` are stored and accessible to smart contracts. Old MPT data is also deleted in accordance with `GarbageCollectionPeriod` setting. |
| ReservedAttributes | `bool` | `false` | Allows to have reserved attributes range for experimental or private purposes. |
| SaveStorageBatch | `bool` | `false` | Enables storage batch saving before every persist. It is similar to StorageDump plugin for C# node. |
| SecondsPerBlock | `int` | `15` | Minimal time that should pass before next block is accepted. |
| SeedList | `[]string` | [] | List of initial nodes addresses used to establish connectivity. |
| StandbyCommittee | `[]string` | [] | List of public keys of standby committee validators are chosen from. |
| StateRootInHeader | `bool` | `false` | Enables storing state root in block header. | Experimental protocol extension! |
| StateSyncInterval | `int` | `40000` | The number of blocks between state heights available for MPT state data synchronization. | `P2PStateExchangeExtensions` should be enabled to use this setting. |
| ValidatorsCount | `int` | `0` | Number of validators set for the whole network lifetime, can't be set if `ValidatorsHistory` setting is used. |
| ValidatorsHistory | map[uint32]int | none | Number of consensus nodes to use after given height (see `CommitteeHistory` also). Heights where the change occurs must be divisible by the number of committee members at that height. Can't be used with `ValidatorsCount` not equal to zero. |
| VerifyBlocks | `bool` | `false` | Denotes whether to verify the received blocks. |
| VerifyTransactions | `bool` | `false` | Denotes whether to verify transactions in the received blocks. |
