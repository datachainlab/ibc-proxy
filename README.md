# IBC Proxy

![Test](https://github.com/datachainlab/ibc-proxy/workflows/Test/badge.svg)
[![GoDoc](https://godoc.org/github.com/datachainlab/ibc-proxy?status.svg)](https://pkg.go.dev/github.com/datachainlab/ibc-proxy?tab=doc)

IBC-Proxy is a module to proxy one or both of the verifications between two chains connected by IBC.

**This software is still under heavy active development.**

## Overview

IBC-proxy provides the following two components:

1. Proxy Client to verify the state of the Proxy and its commitments (also compliant with ICS-02).
2. Proxy Module to verify the state of counterparty chain and generates verifiable commitments using Proxy Client.

In IBC, cross-chain communication is usually achieved by verifying a message from the counterparty chain with a light client and handling it.  Since the required light clients are different for each chain, all chains that intend to communicate with a chain need to implement the corresponding light client as smart contract.

It may be not easy to achieve for some blockchains. The execution environments for smart contracts are diverse, and there are some restrictions on supported languages and constraints on computational resources such as gas prices. In constructing a heterogeneous blockchain network, having the feasible network topology limited by these chain-specific problems is not desirable.

This problem is because the communication destination and the verification destination are combined in the current IBC. Therefore, IBC-Proxy enables the isolation of verification and communication of the counterparty chain. 

- A client on Proxy chain verifies a "upstream" chain, and Proxy module generates a `Proxy Commitment` and its proof corresponding to its verification
- A "downstream" chain uses Proxy Client to verifies the commitment proof instead of verifying the upstream chain's directly.

The following figure shows the concept:

![proxy-packet-relay](./docs/proxy-packet-relay.png "proxy-packet-relay")

The figure shows the case where Proxy P0 verifies Chain C0 and Chain C1 verifies P0. Note that for the reverse direction packet flow, you can select a configuration where C0 verifies C1 directly without Proxy.

## Demo

- ICS-20 through proxy: https://github.com/datachainlab/ibc-proxy-relay/blob/796833cfc0012645079691eaad5fb16f217300c0/.github/workflows/test.yml#L63

## Definitions

`Proxy Machine` (`Proxy` for short) refers to a machine that holds `Proxy Module`.

`Proxy Client` refers to an IBC Client for `Proxy`.

`Downstream` refers to a chain has an IBC Client instance corresponding to `Proxy`.

`Upstream` refers to a chain to be verified by `Proxy`.

## Spec

### Proxy Commitment

Proxy Module verifies the commitment of the Upstream and if successful, provides the Downstream with a commitment on the Proxy that is distinct from the existing commitment of IBC. We call a commitment generated by a Proxy that can be verified by Proxy Client a "Proxy Commitment".

The commitment of IBC must satisfy the property that a specific path binds a unique value as specified in ics23. For this reason, it is necessary to define the specification of the path in which the Proxy stores the commitment to Downstream separately from the format of each commitment path specified in the IBC.

In the definition of the spec, the following points need to be kept in mind:
1. Multiple different downstreams will refer to the same `Proxy`
2. The same `Proxy` provides the proxy for multiple different upstreams
3. There can be multiple prefixes for a given upstream

The type of Proxy Commitment is mapped one-to-one to various status indicators such as Client, Connection, Channel, etc. as in IBC. In IBC, the commitment path is guaranteed to be it by including an unique identifier in a certain host.

However, since the Proxy must use the identifiers on the Upstream, path collisions can occur when different Upstreams are supported on one Proxy.

Therefore, in order to obtain a path that is unique, we introduce a new commitment path format as follows:
`/{proxy_prefix}/{upstream_client_id}/{upstream_prefix}/{upstream_commitment_path}`

- proxy_prefix
    - Proxy store prefix
- upstream_client_id
    - Client ID corresponding to the upstream on the Proxy
- upstream_prefix
    - Store prefix of the upstream
- upstream_commitment_path
    - IBC Commitment path in the upstream

Downstream builds this path based on the state of the Proxy Client and uses it during verification.

## Author

- [Jun Kimura](https://github.com/bluele)
