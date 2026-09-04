# Boards and columns

Resolve project, board, and column IDs before changing them.

## Boards

```text
weeek board list --project ID
weeek board get ID --project ID
weeek board create --name NAME --project ID
weeek board update ID --name NAME
weeek board move ID [--after ID]
weeek board delete ID
```

Project IDs must be positive. `board move` places the board after the supplied
positive board ID; omit `--after` to use the API's default placement.

```sh
weeek project list
weeek board list --project 42
weeek board get 7 --project 42
weeek board create --name "Delivery" --project 42
```

## Columns

```text
weeek column list [--board ID]
weeek column create --name NAME --board ID
weeek column update ID --name NAME
weeek column move ID [--after ID]
weeek column delete ID
```

`column list` can return all visible columns or filter by a positive board ID.
Create requires a positive board ID. `column move --after` requires a positive
column ID when present.

```sh
weeek column list --board 7
weeek column create --name "In review" --board 7
weeek column move 9 --after 8
```

## Place tasks on a board

Use task placement commands after resolving the IDs:

```sh
weeek task location add 123 --project 42 --column 9
```

Before update, move, or delete, list or get the target and disambiguate duplicate
names with the user.
