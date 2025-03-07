### Goal: simulate 1 billion MAU with 1 trillion messages per month
#### Current basic mvp: max throughput 4000msg/s (10B messages per month)

# observability dashboard (grafana + prometheus)

![Grafana](./grafana.png)

# Current (draft) system design
![System design](./architecture.png)


##### lessons from latest testing:
- 2 vcore xcylla can handle 10k ops/s at least
- when scylla can no longer fit all rows in cache read latencies increase from 1ms to 100ms, likely because of COUNT * scan queries for metrics (need to fix this)
- a lot of time is waste in decoding/encoding json -> use binary formats, protobuf + add simple fast compression on top of that
- a lot of time is spent getting all users in a chat, since every time a goroutine wants to notify users it queries db -> lots of the same reads
- a lot of time is spent by scheduler switching from one goroutine to another, a lot of time is spent in sync.Map and mutex synchronizations

##### next immediate tasks:
- [x] ability to test locally + deploy to AWS
- [x] move to ScyllaDB
- [ ] move to zero-copy IO websockets with poller design
- [ ] move ws communication to protobufs
- [ ] move away from architecture where each thread is handling ws_read/logic/queries/ws_write towards an event driven architecture where there are goroutines pools responsible for different steps
- [ ] intra-server event pub/sub with broadcast (assumes 1 cluster for now)

Future:
- right now there are users that receive a msg notification every 100ms, that's unlikely to be realistic, much better testing setup is needed

# basic TODO:
- [ ] save on network prices during stress testing by compressing messages + custom protocol used during stress testing that can duplicate message by a factor of X when transmitting over network to simply save on network traffic ie. msg ABC with factor = 3 becomes ABCABCABC
- [ ] separate websocket communication layer from core logic
- [ ] settle on a load simulation config, and request confirmation metrics
- [ ] put a caching layer and batch db operations
- [ ] add inter server communication and horizontal scaling
- [ ] add message/chat sharding

idk about this:
- [ ] proper chat client, better usability, encryption, safety guruantees, failure recovery, accurate user behaviour simulation