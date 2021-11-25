# IBC Proxy

![Test](https://github.com/datachainlab/ibc-proxy/workflows/Test/badge.svg)
[![GoDoc](https://godoc.org/github.com/datachainlab/ibc-proxy?status.svg)](https://pkg.go.dev/github.com/datachainlab/ibc-proxy?tab=doc)

**This software is still under heavy active development.**

IBC-Proxy is a module to proxy one or both of the verifications between two chains connected by IBC.

IBC-proxy provides the following two components:

1. Proxy Client to verify the state of the Proxy and its commitments (also compliant with ICS-02).
2. Proxy Module to verify the state of counterparty chain and generates verifiable commitments using Proxy Client.

In IBC, cross-chain communication is usually achieved by verifying a message from the counterparty chain with a light client and handling it.  Since the required light clients are different for each chain, all chains that intend to communicate with a chain need to implement the corresponding light client as smart contract.

It may be not easy to achieve for some blockchains. The execution environments for smart contracts are diverse, and there are some restrictions on supported languages and constraints on computational resources such as gas prices. In constructing a heterogeneous blockchain network, having the feasible network topology limited by these chain-specific problems is not desirable.

This problem is because the communication destination and the verification destination are combined in the current IBC. Therefore, IBC-Proxy enables the isolation of verification and communication of the counterparty chain. 

- A client on Proxy chain verifies a "upstream" chain, and Proxy module generates a commitment proof corresponding to its verification
- A "downstream" chain uses Proxy Client to verifies the commitment proof instead of verifying the upstream chain's directly.

The following figure shows the concept:

![proxy-packet-relay](./docs/proxy-packet-relay.png "proxy-packet-relay")

The figure shows the case where Proxy P0 verifies Chain C0 and Chain C1 verifies P0. Note that for the reverse direction packet flow, you can select a configuration where C0 verifies C1 directly without Proxy.

## Author

- [Jun Kimura](https://github.com/bluele)
