# Tutorial

Folder [tutorial](./tutorial/) contains some files from a recipe 
collection that we will work on to demonstrate the core offering of 
the `fm` CLI. 

## Goal

Files got three fields `vegan`, `vegetarian` and `veggie` (synonymous to vegetarian),
most often boolean but inconsistent,
that we will turn into a new list field `diet` to accomodate other
dietary restrictions and preferences (like allergies).

Use `git diff` to see changes applied by each command below.  
Use `git reset --hard HEAD` to undo changes.


## Step 1: Harmonize fields

```sh
fm update tutorial/* set \
  vegan:bool, \
  vegetarian:bool, \
  veggie:bool, \
  diet:list
```

- Creates missing fields, with `null` value.
- `veggie` and `vegan` fields are turned into `false` if zero or blank, everything else `true`, `null` remains.
- Turns any non-list `diet` into a list of length 1.


## Step 2: Append to `diet` list

```sh
fm update tutorial/* set diet+=vegan      where vegan=true
fm update tutorial/* set diet+=vegetarian where vegetarian=true or veggie=true
```


## Step 3: Drop deprecated fields

```sh
fm alter tutorial/* drop vegan, vegetarian, veggie
```
