Checks a list of RSS/Atom feeds for new posts.


## Run

```shell
go run . <command>

# or:

go build --ldflags "-s -w"
./rss-checker <command>
```

## Commands

```shell
add    <feed-url>  # add a feed
remove <feed-url>  # remove a feed
list               # list all feeds
check              # check all feeds for new items
```
