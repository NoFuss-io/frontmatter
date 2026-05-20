#!/bin/bash
# Extract from tutorial.md - quoted command variants

# Step 1: Inspect current state
fm 'select vegan, vegetarian, veggie, diet from tutorial/*'

# Step 2: Normalize fields
fm 'update tutorial/* set vegan:bool, vegetarian:bool, veggie:bool'

# Step 3: Populate `diet` from existing fields
fm 'update tutorial/* set diet+="vegan"      where vegan=true'
fm 'update tutorial/* set diet+="vegetarian" where vegetarian=true or veggie=true'

# Step 4: Verify results
fm 'select vegan, vegetarian, veggie, diet from tutorial/*'

# Step 5: Clean up deprecated fields
fm 'alter tutorial/* drop vegan, vegetarian, veggie'
