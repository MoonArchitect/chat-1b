### Goal: simulate 1 billion MAU with 1 trillion messages per month
#### Current basic mvp: max throughput 30,000 msg/s (80B messages per month)

# observability dashboard (grafana + prometheus)

![Grafana](./grafana.png)

# Current (draft) system design
![System design](./architecture.png)


##### lessons from latest testing:
- 8vcpu (c6gd.2xlarge) scylladb instance can easily handle 30k writes/sec at 40% load => ~500k writes/sec on a c6gd.metal? => 2M+ msgs/sec on a cluster of 5ish instances?
- need better testing setup, better control over load, better user simulation behaviour, better client machine distribution (launch 100 1vcpu instances all over the globe, etc.)
- finer grained profiler tool might be nice
- a lot of time is spent getting all users in a chat, since every time a goroutine wants to notify users it queries db -> lots of the same reads
- a lot of time is spent by scheduler switching from one goroutine to another, a lot of time is spent in sync.Map and mutex synchronizations

##### next immediate tasks:
- [x] ability to test locally + deploy to AWS
- [x] move to ScyllaDB
- [x] move to zero-copy IO websockets with poller design
- [x] move ws communication to protobufs
- [ ] build better chat (msg order, etc., make it like an actual usable chat)
- [ ] build better testing agent + control panel
- [ ] initialize database with 2 months worth of data
- [ ] run actual stress test for future references
- [ ] optimize monolith architecture
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