# Log

This is a simple logging library for Kubernetes. It aims to be a more strict, focused, handy and performant alternative to [Klog](https://github.com/kubernetes/klog).

## Output format

Logs are written in a format very similar to **Kubernetes log format** (unofficial term)

> [!NOTE]
> Format was originally defined in glog (C++): [definition](https://github.com/google/glog/blob/master/src/glog/logging.h#L297), [implementation](https://github.com/google/glog/blob/master/src/logging.cc#L1617).
> Then it was migrated (with slight changes) to glog (Go): [implementation](https://github.com/golang/glog/blob/master/internal/logsink/logsink.go#L204).
> Currently it is defined in KLog ([definition](https://github.com/kubernetes/klog/blob/main/klog.go#L632), [implementation](https://github.com/kubernetes/klog/blob/main/internal/buffer/buffer.go#L117)) as: `Lmmdd hh:mm:ss.uuuuuu threadid file:line] msg...`

### Differences
 - *(incompatible)* In order to support [Contextual Logging](https://kubernetes.io/docs/concepts/cluster-administration/system-logs/#contextual-logging) and make it more explicit in logs, the space-padded fixed-length `threadid` integer field is now a variable-length string consisting of one or more alphanumeric with underscore logger name strings, separated by a semicolon (`:`).
 - *(incompatible)* `file:line` is now optional. In order to locate the exact log line, the logger name together with message should be enough in most cases, so it's reasonable to refuse exposing filesystem names, relying on it in logs and taking the performace hit from this feature. For log parsers, which deal with both cases, it is safe to treat `file:line` as a continuation of the last logger name and assume that `]` ends this sequence.
 - Logs are *always* compatible with **[Structured Logging Format](https://kubernetes.io/docs/concepts/cluster-administration/system-logs/#structured-logging)**, which is a subset of **Kubernetes log format**.
    - Values are *always* double-quoted strings safely escaped with Go syntax.
    - Keys are *always* alphanumeric with underscore strings.
 - Log *always* take up exactly one line .
 - There are always exactly 5 severity levels: `VIWEF`. `V` was added for *Verbose* logs.
 - Time is always in GMT (UTC+0) timezone.

### Format

Log line:
```
{S}{T} {logger1}:{logger2}:{loggerN} {file}:{line}] "{message}" {key1}="{value1}" {key2}="{value2}" {keyN}="{valueN}"
```

---

`{S}` - log severity. One of:
 - `V` - Verbose. The most detailed log.
 - `I` - Informational. Relevant to a regular log reader. Usually a default severity for readers.
 - `W` - Warning. Should bring an attention.
 - `E` - Error. Reports an issue.
 - `F` - Fatal. Reports a critical issue causing application termination.

---

`{T}` - date and time of the log in the format:

```
MMdd hh:mm:ss.uuuuuu
```

Example: `0925 23:46:02.876000`

---

`{logger1}`, `{logger2}`, `{loggerN}` - sequence of logger names. At least one non-empty logger name is specified. Alphanumeric with underscore (`_`) strings.

---

`{file}` - name of the source code file. Alphanumeric string with dot (`.`), underscore (`_`) and hyphen (`-`). Optional. If not present, the preceding space, and the following `:{line}` are omitted as well.

---

`{line}` - line number in `{file}`. Non-negative integer. Only present if `{file}` is present.

---

`{message}` - log message. Double-quoted string safely escaped with Go syntax.

---

`{key1}`, `{key2}`, `{keyN}` - sequence of structured log keys. Alphanumeric with underscore (`_`) strings. Optional. If not present, the preceding space, the following `=` and the corresponding `"{valueN}"` are omitted as well.

---

`{value1}`, `{value2}`, `{valueN}` - sequence of structured log values. Double-quoted string safely escaped with Go syntax.

---


### Example

```
V0125 00:15:15.525108 1:2741:my_service:ba6d2b0b handler.go:116] "received user request" data="This is text with a line break\nand \"quotation marks\"." someField="0.1" someStruct="{StringField:First line,\nsecond \"line\". X:-42 Y:NaN}"
```

## API

TODD

## Compared to klog

## Features

## Usage outside of Kubernetes

Package has no dependency on Kubernetes, and you can absolutely use it in other environments, but there are limitations you should be aware of:

 - Log output format is fixed and there're no plans in making it configurable
 - It's built for containers in mind
   - output goes directly to standard error
   - timezone is GMT (UTC+0)

