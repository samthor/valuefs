# valuefs

## Syntax

Virtual files can be read from valuefs.
These are requested via suffix on a regular file plus a query (currently just duration).
These are well-documented inside [the interface](db/interface.go).

* `#` **Average**
* `%` **Total**
* `@` **ValueAt**
* `^` **SafeLatest**

For example, if a file `ac_temp` exists, you can request its average over the past 10 minutes by reading this file-

`$ cat ac_temp#10m`

Similarly, you could request its total over the past ten minutes, or the value it was >10 minutes ago.
In the case of functions over the data - e.g., average or total - if there are no values within the time period, the file will not exist.
