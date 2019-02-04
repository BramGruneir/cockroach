# Foreign Keys and Cockroach Labs

This tech note discusses the current state of foreign key constraints in Cockroach and tries to
elucidate the potential directions we can go in.

## Goals

- Faster lookups, including in a geo-distributed cluster
- Cascading and lookups that conform to the SQL standard and closely match Postgres

## History and Current Status

Foreign key constraints were conceived and executed on a very clean mental model.  That model
specifically was that any constraints was between one index on the referring table, to an index on
the referenced one.  This allowed not only for a fairly quick development but allowed easy to
understand lookups.  Since both sides must have an index, it's trivial to perform an index scan.
Sadly, the cost of adding a secondary index can be quite high.

### What's wrong?

In a nutshell, foriegn key lookups are slow. And doubly so in partitioned geo-distributed clusters.
There are numerous reasons for this:

- the lookup mechanism is not well batched
- lookups occur on a per row basis instead of per statement or transaction
-

Furthermore, the current model is not flexible enough:

- adding a secondary index to a table incurs a large cost
- not being able to select the lookup index t

## When should an action/lookup occur?

Currently, all foreign key lookups occur after each row in inserted, updated or deleted. Let's call
these collectively mutations.  This is obviously not

## What needs to be done?

### Row vs Statement vs Transaction

As mentioned previously

### Index selection

With geo-distributed clusters in mind.  Having the hard

### Allow multiple constraints on a column

T

### Remove the `index to index` requirement

Foreign key reference descriptors should be moved out of index descriptors and put directly in table
descriptors. Furthermore, they should be defined as a link of between a set of columns on the
referring table to another set of columns on the

They should be moved

## Possible solutions



## Interesting Other Ideas

### Designate a index to be automatically copied to some or all partitions

Instead of creating a collection of indexes, instead add some syntax to designate that there should
be a copy of this index in some or all partitions. Internally, this would look like multiple indexes
but to the end user, it should look like a single index. It will be kept in sync



