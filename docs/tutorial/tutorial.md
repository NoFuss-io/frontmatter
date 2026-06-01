# Tutorial

Folder [recipes](./recipes/) contains sample files from a recipe collection demonstrating `fm` core features.

## Goal

Files have inconsistently-typed dietary fields: `vegan`, `vegetarian`, `veggie` (synonym for vegetarian).
Goal: consolidate into a `diet` list field to support multiple dietary properties.

Use `git diff` to inspect changes from each command. Use `git reset --hard HEAD` to undo.

## Step 1: Inspect current state

```sh
fm 'select vegan, vegetarian, veggie, diet from tutorial/*'
```

View current field values and types across files.

## Step 2: Normalize fields

```sh
fm 'update tutorial/* set vegan:bool, vegetarian:bool, veggie:bool'
```

- Casts each field to specified type. Missing fields created with `null`.
- Errors if a field exists but cannot cast to the target type (file left unchanged).

## Step 3: Populate `diet` from existing fields

```sh
fm 'update tutorial/* set diet+="vegan"      where vegan=true'
fm 'update tutorial/* set diet+="vegetarian" where vegetarian=true or veggie=true'
```

- Where clause evaluates field values: `true`, non-zero numbers, non-null/non-false values are truthy.
- `+=` creates `diet` as a list on first append if missing, then appends subsequent values.
- Single-quote the whole query so the shell passes `diet+="vegan"` verbatim to `fm`; without it the shell strips the inner double quotes and `vegan` is parsed as a field reference instead of a string literal.

## Step 4: Verify results

```sh
fm 'select vegan, vegetarian, veggie, diet from tutorial/*'
```

## Step 5: Clean up deprecated fields

```sh
fm 'alter tutorial/* drop vegan, vegetarian, veggie'
```

Remove old fields now consolidated into `diet`.
