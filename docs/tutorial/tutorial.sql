-- Step 1: Inspect current state
select vegan, vegetarian, veggie, diet from tutorial/*;

-- Step 2: Normalize fields
update tutorial/* set vegan:bool, vegetarian:bool, veggie:bool;

-- Step 3: Populate `diet` from existing fields
update tutorial/* set diet+="vegan"      where vegan=true;
update tutorial/* set diet+="vegetarian" where vegetarian=true or veggie=true;

-- Step 4: Verify results
select vegan, vegetarian, veggie, diet from tutorial/*;

-- Step 5: Clean up deprecated fields
alter tutorial/* drop vegan, vegetarian, veggie;
