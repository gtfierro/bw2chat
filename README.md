Ordo
========

You need an entity w/ permission to publish and subscribe on `gabe.ns/chatrooms/room/+`

```bash
# if you need an entity
bw2 mke -o chatroomentity.ent

# to grant access to a specific chatroom
bw2 mkdot --from <path to granting entity> --to chatroomentity.ent --uri "gabe.ns/chatrooms/room/roomname" --ttl 0 --permissions "PC*"

# to grant access to a *all* chatrooms
bw2 mkdot --from <path to granting entity> --to chatroomentity.ent --uri "gabe.ns/chatrooms/room/+" --ttl 0 --permissions "PC*"

# to grant access to a *all* chatrooms with ability to invite others
bw2 mkdot --from <path to granting entity> --to chatroomentity.ent --uri "gabe.ns/chatrooms/room/+" --ttl <number of invites> --permissions "PC*"
```

TODO: maybe wrap these up in a chatroom-specific tool to simplify?


## Usage

```bash
bw2chat -e chatroomentity.ent client --alias mynamehere
```

Once you are logged in, you get some commands:

```
# join a chatroom
\join roomname 
```

And that's all I've implemented and tested. You can switch between rooms using `\join` and it will keep a log of messages in other rooms.


---

Novus Ordo Seclorum -- Randy Waterhouse
