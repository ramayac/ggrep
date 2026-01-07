# ggrep

A simplified grep-like utility written in Go, designed for recursive searching within files (including traversing `.zip` archives).

## Why?

There is a good story behind this silly tool.

I was working at 'Big Bank in El Salvador', it was late 2013, and I got this situation where I needed to search for specific strings in a lot of log files (and text reports).
Usually we would download the files and examine them by hand in... Notepad++ (yeah it was that bad), but it took ages.
Curious me got ssh access to the ftp server, and while we didn't have 'grep' or 'ack' or ... anything useful... well, Java was there... and I'm mostly a Java developer :)

I decided to make my own grep with Java, nothing fancy, keep it simple and useful (do one thing and do it right), so I made JGrep: https://gist.github.com/ramayac/8926162. I used gist to track the small change I made to it. 
This little piece of code save me a lot of time back then.

Then in 2017, when I started working at 'Big Canada Telco', I landed into a tiny 3 person team, lots of code, lots of meetings and also lots of production support!!!
And I  run into the same scenario, just with a lot of more data... 22 GB of logs daily easy, and then I had the archived logs (zip) too that needed to be searched.

I clearly remember one coworker showing me how did support: downloading log files with filezilla to his personal computer, just to search for the strings he needed using (the excellent I might say) [glog](https://glogg.bonnefon.org/). 
Then get request and response from the services involved in the transaction and send an email to the right team to troubleshoot.

Of course we had the benefit of being 'on site' so download speed was not an issue... he said that anyway, but in reality it was a real pain.
Before I had to do any support (we had a 2 week rotation on it), curious me jump in and did ssh to the ftp server, I log in and see the same restrictions, and only 25 MB of space for the ftp user home, more than enough for my needs. I check for grep, nothing, I check for Java, its there.

Well, lightning *can* strike twice!

I open the gist, build the little .jar, upload and it works like magic.
The little .jar file really saved us alot of time, and might have blocked the server for a couple of seconds once or twice. 
Support dropped from days of downloading files and searching logs to just executing a command and waiting 2 to 5 min at worst to get the results, turns out the server had great performance and a lot of idle time.

My coworker was happy, I was happy, and my BSA did not belive we could do support that fast. Good times!

Now it's 2025, and I like Go, first thing I do? Give new life to the tool that saved me so many hours searching strings in GB of log files.

GGrep _was_ a 1:1 port of JGrep, but a couple of commits ago I decided to just do a more "golang" implementation.
Anyway the point is to have fun, have my tool updated and share it, maybe it will help someone else too.

## Usage

Put ggrep in the folder where your logs files are.
Then run:

```bash
./ggrep [0-9][0-9][0-9] *.log
```

You will get a file called `out.txt` with the results.

### Arguments

*   `regex`: The regular expression or string to search for.
*   `ext` or `--all`: File extension to filter by (e.g., `.log`, `.txt`) or `--all` to search all files.
*   `lines`: (Optional) Number of context lines to display. Defaults to 1.
*   `-s`: (Optional) Silent mode.


## Build

You can build the project using the provided Makefile or standard Go tools:

```bash
make build
```

## Test

Well, just do:

```bash
make test
```
