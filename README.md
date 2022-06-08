# gtools

## Rational

### Run commands defined by a JSON configuration file

execute as:

``` bash
./gtool
```

The default configuration file will be gtool-config.json

```bash
./gtool myConfigFile.json
```

For a specific config file.

## Config data

---

The config file must be valid JSON.

Full definition of all valid fields:

``` json
{
    "config": {
        "showExit1": false,
        "runAtStart":"Start Code",
        "runAtEnd":"Clean up",
        "localConfig": "gtool-config.json",
        "localValues": {
            "commitMessage": {
                "desc": "Commit message",
                "default": "commit",
                "input": true
            }
        }
    },
    "actions": [
        {
            "tab": "tab1",
            "rc": 1,
            "name": "Start Code",
            "desc": "Start the dev environment",
            "hide": false,
            "list": [
                {
                    "cmd": "code",
                    "args": [
                        "."
                    ]
                }
            ]
        }
    ]
}
```

| Field name      | Description | optional |
| ----------- | ----------- | --------- |
| config | Contains global config data | optional |
| config.showExit1 | true \| false Show the Close(1) button. App exit with return code | optional = false |
| config.runAtStart | Run action at startup. If "" then no action is taken | optional = "" |
| config.runAtEnd | Run action before exit. If "" then no action is taken | optional = "" |
| config.localValues | Contains a list of cached fields for substitution in args. See 'Local Values' below | optional |
| config.localConfig | Read additional configuration data from a file. Contents overrieds main file | optional |
| actions | Contains All actions. See Actions below| mandatory |

### Actions

---

Actions defines a list or set of objects that define commands and arguments executed by the os.

Actions can take two forms:

Form 1 is as a JSON list. In this case the actions are displayed in the order they appear in the JSON

```json
"actions": [
    {
        "name": "Start Code",
        ...
    },
    {
        "name": "List directory",
        ...
    },
]
```

Form 2 is as JSON ojects. In this case the actions are displayed in alphabetical order. The objecdt names are ignored otherwise.

```json
"actions": {
    "B Action 1": {
        "name": "Start Code",
        ...
    },
    "A Action 2":  {
        "name": "List directory",
        ...
    },
}
```

Each individual action is defined as follows:

```json
"hide": true,
"rc" : 1,
"name": "Start VSCode",
"desc": "Start the dev environment",
"hide": false
"list": [
    {
        "cmd": "code",
        "args": [
            "."
        ],
        "in": "",
        "inPwName": "",
        "outPwName": "",
        "outFile": "",
        "outFilter": "",
        "errFile": "",
        "delay": 0,
    }
]
```

| Field name | Description | optional |
| ----------- | ----------- | --------- |
| name | Defined the value displayed in the action button | required |
| desc | Defined the value displayed along side the action button | required |
| hide | if true will not display a button. Used with runAtStart | optional = false |
| list | Defines a number of commands to be run one after the other | required |
| rc | Once the list of actions is complete, Exit the application with the return code given | Optional |

### Commands (cmd)

Each command has the following fields:

| Field name | Description | optional |
| ----------- | ----------- | --------- |
| cmd | the command to be run (excluding any arguments). E.g. ls | required |
| args | A String list of arguments. E.g. '-lta' See Agrs below | required |
| in | Input to the sysin stream if a cmd requires it. See In Filters below | optional = "" |
| inPwName | The name of the localValue that holds tha value of the password used to decrypt the 'in' (sysin) stream. Note 'in' cannot be empty. | optional = "" |
| outFile | Output from stdout will be written here. See Output below | optional = "" |
| outPwName | The name of the localValue that holds tha value of the password used to encrypt the 'outFile' (sysout) stream. Note 'outFile' cannot be empty. | optional = "" |
| outFilter | Filter the output using Selects. See Out Filters below | optional = "" |
| errFile | Output from stderr will be written here. See Output below | optional = "" | optional = "" |
| delay | Delay between each cmd in Milli Seconds. 1000 = 1 second| optional = 0 | optional = "" |

### Args

Args are defined as a String list. For example if we wand to execute the command:

```bash
go mod tidy
```

We would define the folllowing json:

```json
cmd": "go",
"args": [
    "mod", "tidy"
],
```

Each argument can contain a substitution expression. This expression will remain unchanged if it's source cannot be found.

There are two sources:

1: The result of a previous 'outFile' where 'memory is defined. For example:

```json
"outFile": "memory:myvar"
```

The sysout from the command is optionally filtered  (see Out Filters below) and stored in a cache with the name 'myvar'

2: The result of a 'localValues' entry defined in the 'config' section of the config file.

```json
"localValues": {
    "commitMessage": {
        "desc": "Commit message",
        "default": "commit"
    }
}
```

If 'myvar' contains the text **'ready to'**.

The result of the following:

```json
cmd": "echo",
"args": [
    "%{myvar}", "%{commitMessage}"
],
```

will be:

```bash
echo ready to commit
>ready to commit
```

### Output

The 'outFile' parameter has multiple forms:

| form | Description |
| ----------- | ----------- |
| A_valid_file_name | The file will be deleted first, overwriting the previous content. Sysout will be written to the file. Only a single result will be written. |
| append:A_valid_file_name | The 'append:' prefix means Sysout will be appended to the file. |
| memory:name_in_cache | The 'memory:' prefix means Sysout will be written to the cache with the name 'name_in_cache'. |

### Filters

Filters can be applied to sysout (outFile) and sysin (in) as required.

Two types of filters exist In Filters and Out Filters. They are defined slightly differently.

### Out Filters

An outFilter filters the generated sysout text. This can be written to sysout (default), a file or to cache memory.

For example:

```json
"outFilter":"l1,s1,p1|n2,s2,p2,d2",
```

A _'filter'_ is divided in to _'selector'_(s) by a '|' char.

Selectors consist of up to 4 _'element'_(s) separated by  a ','

All _'selector'_(s) applied to each to each line in order.

A single _'selector'_ does not require a '|'.

Stage 1

If _'element'_[0] is a number then it is a ZERO based line number _'selector'_. If the line number is >= to the number of lines then the line is NOT selected.

If _'element'_[0] is not a valid integer it is treated as a string _'selector'_. All lines that contain the string will be selected.

Stage 2

If _'element'_[1] is a splitter character. The selected line is split in to N parts.

If _'element'_[1] is "". The selected line is treated as a single part.

Stage 3

_'element'_[2] is a  ZERO based part number. The part is selected. If _'element'_[2] is >= number of parts then no part is selected.

If _'element'_[2] is "". The then the whole line is selected even if _'element'_[1] has a splitter.

Stage 4

Any text after _'element'_[2] and a ',' is treated as a suffix.

Each selected piece of text is returned after being appended with the suffix. If no text is selected then no suffix is appended.  

See the test file 'main_test.go' for examples of filters and their returned values.

| Example | Description |
| ----------- | ----------- |
| "outFilter":"xyz" | All lines containing 'xyz' are output |
| "outFilter":"5" | Only line 5 wil be output. If there are no enough lines, no output will be written |
| "outFilter":"xyz\|4" | All lines containing 'xyz' and line 4 are output |
| "outFilter":"xyz,=,1" | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output |
| "outFilter":"xyz,=,1,," | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output followed by a ',' |
| "outFilter":"xyz,=,1,\n" | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output followed by a new line |
| "outFilter":"xyz,=,1, . " | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output followed by a ' . ' |
| "outFilter":"xyz,=,1,," | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output followed by a ',' |
| "outFilter":"xyz,,,\n" | All lines containing 'xyz' are output followed by a new line |
| "outFilter":"0,,,, \|1,,,\n" | Line 0 is written followed by a ', ' followed by line 1 folllowed by a new line |

### In Filters

The in filter is defined as part of the 'in' value fo the commands in an action list.

| Example Without filters | Description |
| ----------- | ----------- |
| "input this value" | The sysin stream will return each char defined in the field |
| "memory:xxy" | The sysin stream will be returned from the memory cache with the name 'xxy' |
| "file:a/file.txt" | The sysin stream will be returned from the contents of the file 'a/file.txt' |

In filters follow the memory cache name or the file name separated by the '|' character.

The filters function in exactly the same way as the out filters. See above.

| Example With filters | Description |
| ----------- | ----------- |
| "input this value" | This format cannot use a filter! |
| "memory:xxy|filter1,=,1,:|filter2" | The sysin stream will be returned from the memory cache with the name 'xxy' filtered |
| "file:a/file.txt|filter1,=,1,:|filter2" | The sysin stream will be returned from the contents of the file 'a/file.txt' filtered |

### Local Values

---

Cached fields are persistent name value entries that can be substituted in to arguments on the fly.

These are substituted using %{fieldName} data elements.

They can be input at the start of an action or defined in the config data.

``` json
"localValues": {
    "commitMessage": {
        "desc": "Commit message",
        "default": "commit",
        "input": true
    }
}
```

In the above extract 'commitMessage' is the {name} of the field.

| Field name      | Description | optional |
| ----------- | ----------- | --------- |
| localValues.{name}.desc | Description used for field input | required |
| localValues.{name}.default | The default value for the field. This is updated if the field is input in the UI | required |
| localValues.{name}.input | true \| false. Input the field in a dialog (once) before running the action | optional=false |
| localValues.{name}.minLen | Input the field in a dialog (once) with a minimum length | optional |
| localValues.{name}.isPassword | Input the field in a dialog (once) treated as a password | optional |

### Encryption and Decryption

A local value is required for encryption and decryption. The name if refered to in the 'in' or 'outFile' definition.

``` json
"localValues": {
    "myPw1": {
        "desc": "Password 1",
        "input": true,
        "value": "",
        "minLen": 5,
        "isPassword": true
    }
}
```

### Encryption (outFile)

The system output will be written to a file and encrypted using the password (key) defined in the local value.

``` json
{
    "cmd": "git",
    "args": [
        "config",
        "-l"
    ],
    "outFile": "encFile.txt",
    "outPwName": "myPw1"
}
 ```

The output from the command 'git config -l' will be writen to the 'encFile.txt' encrypted with the password defined in local value 'myPw1'.

Before the command is run a password entry dialog will be presented for entry of the password. Once entered the value is retained for all further use of the local value 'myPw1'.

### Decryption (in)

The system input (sysin) will be read from file 'encdFile.txt' and decrypted using the password (key) defined in the local value.

``` json
{
    "name": "Read Enc",
    "desc": "Read an encrypted file",
    "list": [
        {
            "cmd": "cat",
            "args": [],
            "in": "file:encdFile.txt"
        }
    ]
}
```

Before the command is run a password entry dialog will be presented for entry of the password. Once entered the value is retained for all further use of the local value 'myPw1'.
