# Projects and portfolios

## Projects

```text
weeek project list
weeek project get ID
weeek project create --name NAME --private 0|1 [--logo URL] [--description TEXT] [--portfolio ID]
weeek project update ID --name NAME --private 0|1 [--logo URL] [--color COLOR]
weeek project archive ID
weeek project unarchive ID
weeek project delete ID
```

Create and update require both `--name` and `--private`; privacy is `0` or `1`.
The optional portfolio ID must be positive.

```sh
weeek project list
weeek project get 42
weeek project create --name "CLI" --private 0 --description "weeek-cli work"
weeek project archive 42
```

Archive when the project may be needed again; delete only when the user explicitly
requests deletion and the ID has been confirmed.

## Portfolios

```text
weeek portfolio list [--search TEXT] [--parent ID] [--limit N] [--offset N]
weeek portfolio get ID
weeek portfolio create --name NAME [--parent ID]
weeek portfolio update ID --name NAME
weeek portfolio delete ID
```

Parent IDs must be positive. List limit is 0 through 100 and offset must not be
negative.

```sh
weeek portfolio list --search "engineering" --limit 20
weeek portfolio create --name "Platform"
weeek portfolio create --name "CLI tools" --parent 5
```

## Resolve hierarchy before changes

```sh
weeek portfolio list --search "platform"
weeek project list
weeek board list --project 42
```

Use the IDs returned by these commands for portfolio, project, board, and task
operations. Ask the user to choose when names are ambiguous.
