[TOC]
## How can I read a file (data stream, variable) line-by-line (and/or field-by-field)?
[不要试着用 "for"](http://mywiki.wooledge.org/DontReadLinesWithFor).  使用 while 和 break:

```bash
while IFS= read -r line; do
  printf '%s\n' "$line"
done < "$file"
```
The `-r` option to `read` prevents backslash interpretation (usually used as a backslash newline pair, to continue over multiple lines or to escape the delimiters).  Without this option, any unescaped backslashes in the input will be discarded.  You should almost always use the `-r` option with read.

The most common exception to this rule is when -e is used, which uses Readline to obtain the line from an interactive shell. In that case, tab completion will add backslashes to escape spaces and such, and you do not want them to be literally included in the variable. This would never be used when reading anything line-by-line, though, and -r should always be used when doing so.

In the scenario above `IFS=` prevents [trimming of leading and trailing whitespace](http://mywiki.wooledge.org/#Trimming). Remove it if you want this effect.

`line` is a variable name, chosen by you.  You can use any valid shell variable name there.

The [redirection](http://mywiki.wooledge.org/BashGuide/InputAndOutput#Redirection) `< "$file"` tells the `while` loop to read from the file whose name is in the variable `file`.  If you would prefer to use a literal pathname instead of a variable, you may do that as well.  If your input source is the script's standard input, then you don't need any redirection at all.

If your input source is the contents of a variable/parameter, [[BASH]] can iterate over its lines using a "here string":

```bash
while IFS= read -r line; do
  printf '%s\n' "$line"
done <<< "$var"
```

The same can be done in any Bourne-type shell by using a "here document" (although `read -r` is POSIX, not Bourne):

```bash
while IFS= read -r line; do
  printf '%s\n' "$line"
done <<EOF
$var
EOF
```

If avoiding comments starting with `#` is desired, you can simply skip them inside the loop:
```bash
# Bash
while read -r line; do
  [[ $line = \#* ]] && continue
  printf '%s\n' "$line"
done < "$file"
```

If you want to operate on individual fields within each line, you may supply additional variables to {{{read```:

```bash
# Input file has 3 columns separated by white space (space or tab characters only).
while read -r first_name last_name phone; do
  # Only print the last name (second column)
  printf '%s\n' "$last_name"
done < "$file"
```

If the field delimiters are not whitespace, you can set [IFS (internal field separator)](http://mywiki.wooledge.org/IFS):

```bash
# Extract the username and its shell from /etc/passwd:
while IFS=: read -r user pass uid gid gecos home shell; do
  printf '%s: %s\n' "$user" "$shell"
done < /etc/passwd
```
For tab-delimited files, use [[Quotes|IFS=$'\t']] though beware that multiple tab characters in the input will be considered as *'one''' delimiter (and the Ksh93/Zsh `IFS=$'\t\t'` workaround won*t work in Bash).

You do *not'* necessarily need to know how many fields each line of input contains.  If you supply more variables than there are fields, the extra variables will be empty.  If you supply fewer, the last variable gets "all the rest" of the fields after the preceding ones are satisfied.  For example,

```bash
read -r first last junk <<< 'Bob Smith 123 Main Street Elk Grove Iowa 123-555-6789'

# first will contain "Bob", and last will contain "Smith".
# junk holds everything else.
```

Some people use the throwaway variable `_` as a "junk variable" to ignore fields.  It (or indeed any variable) can also be used more than once in a single `read` command, if we don't care what goes into it:

```bash
read -r _ _ first middle last _ <<< "$record"

# We skip the first two fields, then read the next three.
# Remember, the final _ can absorb any number of fields.
# It doesn't need to be repeated there.
```

Note that this usage of `_` is only guaranteed to work in Bash. Many other shells use `_` for other purposes that will at best cause this to not have the desired effect, and can break the script entirely. It is better to choose a unique variable that isn't used elsewhere in the script, even though `_` is a common Bash convention.

[TOC]The {{{read``` command modifies each line read; by default it [removes all leading and trailing whitespace]] characters (spaces and tabs, if present in [[IFS](http://mywiki.wooledge.org/BashFAQ/067)). If that is not desired, the {{{IFS``` variable has to be cleared:

```bash
# Exact lines, no trimming
while IFS= read -r line; do
  printf '%s\n' "$line"
done < "$file"
```

One may also read from a command instead of a regular file:

```bash
some command | while IFS= read -r line; do
  printf '%s\n' "$line"
done
```
This method is especially useful for processing the output of [find](http://mywiki.wooledge.org/UsingFind) with a block of commands:

```bash
find . -type f -print0 | while IFS= read -r -d '' file; do
    mv "$file" "${file// /_}"
done
```
This reads one filename at a time from the `find` command and [renames the file](http://mywiki.wooledge.org/BashFAQ/030), replacing spaces with underscores.

Note the usage of `-print0` in the `find` command, which uses NUL bytes as filename delimiters; and {{{-d ''``` in the `read` command to instruct it to read all text into the `file` variable until it finds a NUL byte. By default, `find` and `read` delimit their input with newlines; however, since filenames can potentially contain newlines themselves, this default behaviour will split up those filenames at the newlines and cause the loop body to fail. Additionally it is necessary to set `IFS` to an empty string, because otherwise `read` would still strip leading and trailing whitespace. See [FAQ #20](http://mywiki.wooledge.org/BashFAQ/020) for more details.

Using a pipe to send `find`'s output into a while loop places the loop in a SubShell and may therefore cause problems later on if the commands inside the body of the loop attempt to set variables which need to be used after the loop; in that case, see [FAQ 24](http://mywiki.wooledge.org/BashFAQ/024), or use a ProcessSubstitution like:

```bash
while IFS= read -r line; do
  printf '%s\n' "$line"
done < <(some command)
```

If you want to read lines from a file into an [array](http://mywiki.wooledge.org/BashFAQ/005), see [FAQ 5](http://mywiki.wooledge.org/BashFAQ/005).

### My text files are broken!  They lack their final newlines!

If there are some characters after the last line in the file (or to put it differently, if the last line is not terminated by a newline character), then `read` will read it but return false, leaving the broken partial line in the `read` variable(s). You can process this after the loop:

```bash
# Emulate cat
while IFS= read -r line; do
  printf '%s\n' "$line"
done < "$file"
[[ -n $line ]] && printf %s "$line"
```

Or:

```bash
# This does not work:
printf 'line 1\ntruncated line 2' | while read -r line; do echo $line; done

# This does not work either:
printf 'line 1\ntruncated line 2' | while read -r line; do echo "$line"; done; [[ $line ]] && echo -n "$line"

# This works:
printf 'line 1\ntruncated line 2' | { while read -r line; do echo "$line"; done; [[ $line ]] && echo "$line"; }
```
The first example, beyond missing the after-loop test, is also missing quotes. See [Quotes](http://mywiki.wooledge.org/Quotes) or [Arguments](http://mywiki.wooledge.org/Arguments) for an explanation why. The [Arguments](http://mywiki.wooledge.org/Arguments) page is an especially important read.

For a discussion of why the second example above does not work as expected, see [FAQ #24](http://mywiki.wooledge.org/BashFAQ/024).

Alternatively, you can simply add a logical OR to the while test:
```bash
while IFS= read -r line || [[ -n $line ]]; do
  printf '%s\n' "$line"
done < "$file"

printf 'line 1\ntruncated line 2' | while read -r line || [[ -n $line ]]; do echo "$line"; done
```

### How to keep other commands from "eating" the input
Some commands greedily eat up all available data on standard input.  The examples above do not take precautions against such programs.  For example,
```bash
while read -r line; do
  cat > ignoredfile
  printf '%s\n' "$line"
done < "$file"
```
will only print the contents of the first line, with the remaining contents going to "ignoredfile", as `cat` slurps up all available input.

One workaround is to use a numeric FileDescriptor rather than standard input:
```bash
# Bash
while IFS= read -r -u 9 line; do
  cat > ignoredfile
  printf '%s\n' "$line"
done 9< "$file"

# Note that read -u is not portable to every shell. Use a redirect to ensure it works in any POSIX compliant shell:
while IFS= read -r line <&9; do
  cat > ignoredfile
  printf '%s\n' "$line"
done 9< "$file"
```

Or:

```bash
exec 9< "$file"
while IFS= read -r line <&9; do
  cat > ignoredfile
  printf '%s\n' "$line"
done
exec 9<&-
```

This example will wait for the user to type something into the file {{{ignoredfile``` at each iteration instead of eating up the loop input.

You might need this, for example, with `mencoder` which will accept user input if there is any, but will continue silently if there isn't.  Other commands that act this way include `ssh` and `ffmpeg`.  Additional workarounds for this are discussed in [FAQ #89](http://mywiki.wooledge.org/BashFAQ/089).

----
Pike.SZ.fish
