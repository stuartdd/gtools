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
        "runAtStartDelay":100,
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
| config.runAtStartDelay | Run action at startup after n milliseconds | optional = 500 |
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
        "errFile": "",
        "delay": 0,
        "ignoreError":true
    }
]
```

| Field name | Description | optional |
| ----------- | ----------- | --------- |
| name | Defined the value displayed in the action button | required |
| desc | Defined the value displayed along side the action button | optional = "" |
| hide | If contains '%{' or 'yes' then don't show | optional = "" |
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
| errFile | Output from stderr will be written here. See Output below | optional = "" |
| delay | Delay between each cmd in Milli Seconds. 1000 = 1 second| optional = 0 | optional = "" |
| ignoreError | Dont fail the action if the command fails | Optional=false |

### Args

Args are defined as a String list. For example if we want to execute the command:

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

There are three sources:

1: The result of a previous 'outFile' where 'memory:' is defined. For example:

```json
"outFile": "memory:myvar"
```

The sysout from the command is optionally filtered  (see Out Filters below) and stored in a cache with the name 'myvar'

2: The result of a 'localValues' entry defined in the 'config' section of the config file.

```json
"localValues": {
    "commitMessage": {
        "desc": "Commit message",
        "default": "commit",
        "input": true,
        "minLen": 5
    }
}
```

3: Environment variables

So %{HOME} will return the path to your home directory.

If the cache 'myvar' contains the text **'ready to'**.

If the local variable 'commitMessage' contains the string **'commit'** (as defined above)

If your user id is **'fred'**

NOTE: _You will be prompted for the value of 'commitMessage' as it is an 'input' and you will have to enter at least 5 characters. The default value will be 'commit'_

The result of the following:

```json
cmd": "echo",
"args": [
    "%{myvar}", "%{commitMessage}", "%{HOME}"
],
```

will be:

```bash
echo ready to commit /home/fred
>ready to commit /home/fred
```

### Output

---

The 'outFile' and 'errFile' parameters have multiple forms:

The 'outFile' and 'errFile' parameters can also be followed by a filter:

``` json
"outFile":"memory|gitAuthData|AuthData,=,1"
```

The above will write sysout to memory with the name 'gitAuthData'. It will be filtered (see Filters below) to lines containing 'AuthData' split in to an array at the '=' character selecting the second array element (arrays are zero based).

| form | Description |
| ----------- | ----------- |
| A_valid_file_name * | The file will be deleted first, overwriting the previous content. Sysout will be written to the file. Only a single result will be written. |
| append:A_valid_file_name * | The 'append:' prefix means Sysout will be appended to the file. |
| memory:name_in_cache * | The 'memory:' prefix means Sysout will be written to the cache with the name 'name_in_cache'. |
| clip:name_in_cache | The 'clip:' prefix means Sysout will be written to the cache with the name 'name_in_cache' and also copied to the clipboard. |
| http:URL ** | The 'http:' prefix means Sysout will be written via HTTP POST ans a 'text/plain' mime type to the given URL |
| errFile | Output from stderr will be written here. See Output below | optional = "" | optional = "" |

Note * items apply to 'errFile' as well. 'errFile' definitions cannot be used with encryption, 'clip:' or 'http:'.

### Example http GET and POST

Note that '%{USER}' will be substituted for the uesr id in environment variable USER.

GET:

``` json
{
    "cmd": "cat",
    "in": "http:http://131.200.0.23:8080/files/name/%{USER}.git.data",
    "args": [],
    "outFile": "textfile.txt"
}
```

The above will cat the 'in' stream and write it to sysout. The outFile definition writes sysout to 'textfile.txt'. Asuming that that URL server implements GET data protocol, the received data will be written to the file.

POST:

``` json
{
    "cmd": "cat",
    "args": ["textfile.txt"],
    "outFile": "http:http://131.200.0.23:8080/files/name/%{USER}.git.data"
}
```

The above will 'cat' the file 'textfile.txt' to sysout. The outFile will redirect sysout to the 'http' URL. Asuming that that URL server implements POST data protocol, the file contents will be sent to it.

The file mime type is always assumed to be 'text/plain'.

### In and Out Filters

---

Filters can be applied to sysout (outFile) and sysin (in) as required.

``` json
"outFile":"memory|abc123|user,=,1"
```

Write sysout to memory with the name 'abc123'. The filter will include lines that contain 'user'. Each line will be split by '=' and the [1] element will be output.

So if the line contains "user=stuart". Then 'stuart' will be writen to memory.

``` json
"in":"file:infile.txt|user,=,1"
```

Will read sysin from the file 'infile.txt' and filter the content to include lines that contain 'user'. Each line will be split by '=' and the [1] element will be output.

### Filters

A Filter can filters the generated sysout/syserr text as well a filter in sysin content.

For example:

```json
"outFile":"outfile.txt|l1,s1,p1|n2,s2,p2,d2"
```

A _'filter'_ follows an 'in' or 'outFile' descriptor separated by '|' 

A _'filter'_ is divided in to _'selector'_(s) by a '|' char.

Selectors consist of up to 4 _'element'_(s) separated by  a ','

All _'selector'_(s) are applied to each to each line in order.

A _'selector'_ does not require a '|' at the end.

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
| "xyz" | All lines containing 'xyz' are output |
| "5" | Only line 5 wil be output. If there are no enough lines, no output will be written |
| "xyz\|4" | All lines containing 'xyz' and line 4 are output |
| "xyz,=,1" | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output |
| "xyz,=,1,," | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output followed by a ',' |
| "xyz,=,1,\n" | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output followed by a new line |
| "xyz,=,1, . " | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output followed by a ' . ' |
| "xyz,=,1,," | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output followed by a ',' |
| "xyz,,,\n" | All lines containing 'xyz' are output followed by a new line |
| "0,,,, \|1,,,\n" | Line 0 is written followed by a ', ' followed by line 1 folllowed by a new line |

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
| localValues.{name}.isFileName | Input the field in a dialog (once) treated as a file name | optional |
| localValues.{name}.isFileWatch | Will return a value is the file exists | optional |


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

The sysout will be written to a file and encrypted using the password (key) defined in the local value 'myPw1'.

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
            "in": "file:encdFile.txt",
            "inPwName": "myPw1"
        }
    ]
}
```

Before the command is run a password entry dialog will be presented for entry of the password. Once entered the value is retained for all further use of the local value 'myPw1'.
