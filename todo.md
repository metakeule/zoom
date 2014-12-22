# generelles konzept für prozesse, client -> server kommunikation
  prozess-aufruf kommunikation und bibliothekeneinbindung etc.

1. man kann jede art von datenaustausch als funktionsaufruf oder methodenaufruf betrachten
2. mögliche werte und typen die dabei übergeben werden können, sind
    2.1 basis typen
        2.1.1 string
        2.1.2 int64
        2.1.3 float64
        2.1.4 time.Time
        //2.1.5 io.Reader (stdin)
        //2.1.6 io.Writer (stdout)
        2.1.7 error (stderr)
        2.1.8 filepath
    2.2 slice/array typen
        2.2.1 []string
        2.2.2 []int64
        2.2.3 []float64
        2.2.4 []time.Time
        //2.2.5 []io.Reader
        //2.2.6 []io.Writer
        2.2.7 []error
        2.2.8 []byte
        2.2.9 []filepath
    2.3 map typen
        2.3.1 map[string]string
        2.3.2 map[string]int64
        2.3.3 map[string]float64
        2.3.4 map[string]time.Time
        //2.3.5 map[string]io.Reader (stdin)
        //2.3.6 map[string]io.Writer (stdout)
        2.3.7 map[string]error (stderr)
        2.3.8 map[string][]string
        2.3.9 map[string][]int64
        2.3.10 map[string][]float64
        2.3.11 map[string][]time.Time
        //2.3.12 map[string][]io.Reader
        //2.3.13 map[string][]io.Writer
        2.3.14 map[string][]error
        2.3.15 map[string][]byte
        2.3.16 map[string]filepath
    2.4 structs, die als werte eine der oben angegebenen typen beinhalten
    2.5 optionale typen: pointer zu den oben angegebenen typen

3. bei einem befehl in der commandozeile wird am ende eine methode aufgerufen, die folgendes interface implementiert:

```
    type Command interface {
        Run(stdin io.Reader) (stdout io.Writer, error)
    }
```

um eine function dazu umzuwandeln:

```
    type CommandFunc func (stdin io.Reader) (stdout io.Writer, error)

    func (c CommandFunc) Run(stdin io.Reader) (stdout io.Writer, error) {
        return c(stdin)
    }
```

die parameter werden als werte aus dem struct gezogen, auf dem die methode definiert ist:

```
    type Print struct {
        What string `help:"hier der hilfstext"`
    }

    func (p Print) Doit(stdin io.Reader) (stdout io.Writer, error) {
        ....
    }

    commands.Register(Print{}.Doit)
```

register muss den Print typen herausfinden und die optionen auswerten

# changes

1. separate relations => references and nodes/properties

    type Node struct {
        UUID string 
        Shard string
        props map[string]interface{} // saved in nodes file
        texts map[string]string // saved in each file for a text (text is string lenghth > 255) texts are always UTF-8, \n
        blobs map[string]io.ReadSeeker // saved outside the repo inside the working dir (will be synced via rsync), blobpath must begin with mimetype
    }

    // saved as 
    `node/shard/UUID[0:2]/UUID[2:]` (props)
    `text/shard/UUID[0:2]/UUID[2:]/textname` (texts)
    `blob/shard/UUID[0:2]/UUID[2:]/blobpath` (blobs, saved inside working dir, but outside repo)

    .gitignore contains `blob` and `index`

    type Edge {
        Category string
        From *Node
        To *Node
        Properties *Node
        Weight float64
    }

    // saved as
    `refs/category/shard/From-UUID[0:2]/FromUUID[2:]` (here all references from From (for the given category) are saved as map[string]string, where key is to and val is uuid of properties node (may be empty string for Refs without property node))

    - references are never changed, just added or deleted (no integrity)
    - property nodes may be changed or deleted at any time (no integrity)

   
2. performance and storage

    the data is materialized in a directory on a harddisk. that is the whole database. therein is a blob dir that is ignored by git and a .git directory
    that holds the rest of the data

    the repo (without blobs) must fit in memory

    when the db-server is starting, the repo is cloned (--bare) in to a memory mounted directory.

    all git commands apply to this directory. 

    from time to time the repo is pushed to the original db dir on the harddisk.

    however blobs are always directly saved to and read from the original db dir on the harddisk. they never go to memory


3. sharding + synchronisation (future)

    each shard is a branch. there is no master branch. the working branch is the shard name of the instance, the other shards are remote branches that
    may be pulled to and pushed from the working branch 

    each node belongs to a shard that is part of his storage dir. 
    each node has one masterserver through which it can be created, deleted and modified. however all servers may read all nodes and all shards may push and pull to each other, since there is no path that 2 servers may write to. this way all servers may be synchronized via push and pull.

    synchronization takes place via the original db dir on the harddisk by a separate process.
    the in memory clone will push to it anyway. from time to time the in memory
    clone will pull from the original db dir.

    blobs also belong to shards via their nodes. therefor we can easily synchronize them via rsync

    data should be consistent within one shard and eventually consistent between shards

    since references between shards are possible ensuring their consistency
    will require network tripps

    however, synchronization could be enforced based on which shard was changed.

    a good strategy would be to make a separate shard for data that is
    seldom changing (will not be changed by customers) and needed everywhere
    and then pushing from this shard whenever the data changes and have a
    hook in the receiving harddisk repo that pushes back to the in memory repo

    the same could be done for blobs that are often needed and do seldom change. the shard directory of the corresponding shard could be synced more often via rsync

    the other data should be sharded by a good strategy (e.g. by country) to
    avoid cross shard references between these other shards.

    also it could be a good idea to have separate shards for nodes that have
    long standing transactions (to avoid blocking, since gitdb in single threaded).

    since the shard is part of the node properties it is always clear, which shard to ask for the data. so there could be a inter server locks that are synced between the in memory dirs of the different servers (but not part of the repo) to ensure consistency when needed, e.g.

    `locks/node/shard/UUID[0:2]/UUID[2:]`
    and
    `locks/ref/category/shard/From-UUID[0:2]/FromUUID[2:]`

    that have a map[string]time.Duration where the key is the lock holding shard and the value is the time after which a timeout is considered to have happened and the lock may be removed by the owning shard. the owning
    shard checks, if the lock is still there after duration has passed, deleting the lock eventually. requests to change a corresponding node or 
    ref in the meantime will be queued and executed when the lock went away, so
    that not affected queries can proceed

    we need a special protocol to handle this situation

    we could give each transaction a timeout (as parameter). then if a transaction needs to lock a node/ref from a different shard, the responsible shard will be asked for the lock and given the duration.
    if a lock already exists, the responsible shard will answer with "lock held, try in [time.duration]" where [time.duration] is the time left until the lock would timeout. the asking server will check if the duration lies with the timeout of its transaction and if it does not, the transaction fails, if it does, he will retry after the duration has passed.

    if the lock could be held by the caller, then responsible shard confirms the lock and the duration. the caller then proceeds with his transaction.
    just before the transaction is committed, the caller requests the responsible shard to release the lock. if the lock was still there and held by the caller, the responsible shard confirms the release and removes the lock. then the callers transaction succeeds. otherwise the responsible shard returns an error and the callers transaction fails.

    if the callers transaction fails, it sends a notification to all responsible shards to release the locks he held. the response of that notification is not relevant.

    this communication between servers could be made by websockets. the last mentioned release notification could be a publisher channel each shard has (that is subscribed by each other shard) and then there would be a direct connection between each shard so that everybody knows, if one shards goes down.

4. hot standby and backup (far future)
    
    since it is easy to replicate, each shard could have one or more backup shards to which here pushes regularily. one of these backup shards could
    become active as replacement for the old one (first wins). there must be a protocol for the consense finding, which server is responsible for which shard