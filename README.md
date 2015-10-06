# valuefs

valuefs is a FUSE-based virtual filesystem for numeric values over time.

## Syntax

To write values, simply echo numbers to arbitrary filenames within a mounted FS.
For example-

```bash
echo 1 > power_on
echo 25.45 > house_temp
echo -100 > pending_stuff
echo 24.07 > house_temp
```

The most recent values can be read back, or files removed to clear them.
The filesystem only supports numbers - not strings or arbitrary bytes.

### Virtual Files

valuefs shines because of its virtual file support.
These are read via suffix on a regular file plus a duration query.
These are well-documented inside [the interface](db/interface.go).

* `#` **Average**
* `%` **Total**
* `@` **ValueAt**
* `^` **SafeLatest**

For example, if a file `ac_temp` exists, you can request its average over the past 10 minutes by reading this file-

`$ cat ac_temp#10m`

Similarly, you could request its total over the past ten minutes, or the value it was >10 minutes ago.
In the case of functions over the data - e.g., average or total - if there are no values within the time period, the file will not exist.

Each result has a unique inode.
The value for a single file (e.g., `ac_temp#10m`) tends to be cached for a short period of time, but seemingly at most 30s-1m.
If you're using valuefs as part of a cron or uploader job, don't read too often.

## Logging

For now, valuefs can just dump logs to a flat file.

## License

This work is made available under an Apache2 license.
