# Tutorial

Folder [tutorial](./tutorial/) contains some files from a recipe 
collection that we will work on to demonstrate the core offering of 
the `fm` CLI. 

## Goal

Files got two properties `vegan` and `veggie`,
most often boolean but inconsistent,
that we will turn into a new list field `diet` to accomodate other
dietary restrictions and preferences (like allergies).

Use `git diff` to see changes applied by each command below.  
Use `git reset --hard HEAD` to undo changes.


## Step 1: Harmonize fields

```sh
fm update tutorial/* set veggie:bool, vegan:bool, diet:list
```

- Creates missing properties, with `null` value.
- `veggie` and `vegan` fields are turned into `false` if zero or blank, into `false`, everything else `true`, `null` remains.
- Turns any non-list `diet` into a list of length 1.


## Step 2: Append to `diet` list

```sh
fm update tutorial/* set diet+=vegan      where vegan
fm update tutorial/* set diet+=vegetarian where veggie
```


## Step 3: Drop deprecated fields

```sh
fm alter tutorial/* drop veggie, vegan
``
