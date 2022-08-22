# gtools

## TO-DO
* Run action in background
* Run timed action

## Rational

### Run commands defined by a JSON configuration file

execute as:

``` bash
./gtool
```

The default configuration file will be gtool-config.json in the home dir and if not found there then in the current directory.

```bash
./gtool 
```

For a specific config file.

```bash
./gtool -c myConfigFile.json
```

To log debug data use the -l option

```bash
./gtool -l myLogFile.log
```

## Config data

---

The config file must be valid JSON.

Full definition of all valid fields:

``` json
{
    "config": {
        "showAltExit": false,
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
            "hide": "no",
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
| config.altExit | Defines an optional alternate close button | optional |
| config.altExit.title | The button text | required if altExit defined |
| config.altExit.rc | The OS return code. Allows the calling script to take a specific action | required if altExit defined |
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
"rc" : 1,
"name": "Start VSCode",
"desc": "Start the dev environment",
"hide": "yes",
"list": [
    {
        "cmd": "code",
        "args": [
            "."
        ],
        "path": "",
        "stdin": "",
        "inPwName": "",
        "outPwName": "",
        "stdout": "",
        "stderr": "",
        "delay": 0,
        "ignoreError":true
    }
]
```

| Field name | Description | optional |
| ----------- | ----------- | --------- |
| name | Defined the value displayed in the action button | required |
| desc | Defined the value displayed along side the action button | optional = "" |
| hide | If contains '%{' or 'yes' then don't show. See 'Hide Actions' below | optional = "" |
| list | Defines a number of commands to be run one after the other | required |
| rc | Once the list of actions is complete, Exit the application with the return code given | Optional |

### Commands (cmd)

Each command has the following fields:

| Field name | Description | optional |
| ----------- | ----------- | --------- |
| cmd | the command to be run (excluding any arguments). E.g. ls | required |
| args | A String list of arguments. E.g. '-lta' See Agrs below | optional |
| path | The directory to start the command in (it's existance is NOT checked before running the cmd) | optional = current dir |
| stdin | Defines the stdin stream if a cmd requires it. See In Filters below | optional = "" |
| inPwName | The name of the localValue that holds tha value of the password used to decrypt the 'stdin' stream. Note 'stdin' cannot be empty if inPwName is defined. | optional = "" |
| stdout | Output from stdout will be written here. See Output below | optional = "" |
| outPwName | The name of the localValue that holds tha value of the password used to encrypt the 'stdout' stream. Note 'stdout' cannot be empty. | optional = "" |
| stderr | Output from stderr will be written here. See Output below | optional = "" |
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

1: The result of a previous 'stdout' where 'memory:' is defined. For example:

```json
"stdout": "memory:myvar"
```

The stdout from the command is optionally filtered  (see Out Filters below) and stored in a cache with the name 'myvar'

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

The 'stdout' and 'stderr' parameters have multiple forms:

The 'stdout' and 'stderr' parameters can also be followed by a filter:

``` json
"stdout":"memory|gitAuthData|AuthData,=,1"
```

The above will write stdout to memory with the name 'gitAuthData'. It will be filtered (see Filters below) to lines containing 'AuthData' split in to an array at the '=' character selecting the second array element (arrays are zero based).

| form | Description |
| ----------- | ----------- |
| A_valid_file_name * | The file will be deleted first, overwriting the previous content. Sysout will be written to the file. Only a single result will be written. |
| append:A_valid_file_name * | The 'append:' prefix means Sysout will be appended to the file. |
| memory:name_in_cache * | The 'memory:' prefix means Sysout will be written to the cache with the name 'name_in_cache'. |
| clip:name_in_cache | The 'clip:' prefix means Sysout will be written to the cache with the name 'name_in_cache' and also copied to the clipboard. |
| http:URL | The 'http:' prefix means Sysout will be written via HTTP POST and a 'text/plain' mime type to the given URL |
| stderr | Output from stderr will be written here. See Output below | optional = "" | optional = "" |

Note * items apply to 'stderr' as well. 'stderr' definitions cannot be used with encryption, 'clip:' or 'http:'.

### Example http GET and POST

---

Note that '%{USER}' will be substituted for the uesr id in environment variable USER.

GET:

``` json
{
    "cmd": "cat",
    "stdin": "http:http://131.200.0.23:8080/files/name/%{USER}.git.data",
    "args": [],
    "stdout": "textfile.txt"
}
```

The above will cat the 'stdin' stream and write it to stdout. The 'stdout' definition writes stdout to 'textfile.txt'. Asuming that that URL server implements GET data protocol, the received data will be written to the file.

POST:

``` json
{
    "cmd": "cat",
    "args": ["textfile.txt"],
    "stdout": "http:http://131.200.0.23:8080/files/name/%{USER}.git.data"
}
```

The above will 'cat' the file 'textfile.txt' to stdout. The 'stdout' definition will redirect stdout to the 'http' URL. Asuming that that URL server implements POST data protocol, the file contents will be sent to it.

The file mime type is always assumed to be 'text/plain'.

### In and Out Filters

---

Filters can be applied to stdout and stdin as required.

``` json
"stdout":"memory|abc123|user,=,1"
```

Write stdout to memory with the name 'abc123'. The filter will include lines that contain 'user'. Each line will be split by '=' and the [1] element will be output.

So if the line contains "user=stuart". Then 'stuart' will be writen to memory.

``` json
"stdin":"file:infile.txt|user,=,1"
```

Will read stdin from the file 'infile.txt' and filter the content to include lines that contain 'user'. Each line will be split by '=' and the [1] element will be output.

### Filters

A Filter can filter the generated stdout/stderr text as well a filter in stdin content.

For example:

```json
"stdout":"outfile.txt|l1,s1,p1|n2,s2,p2,d2"
```

A _'filter'_ follows a 'stdin' or 'stdout' descriptor separated by '|' 

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
| "5" | Only line 6 (because ZERO based) wil be output. If there are no enough lines, no output will be written |
| "xyz\|4" | All lines containing 'xyz' and line 4 are output |
| "xyz,=,1" | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output |
| "xyz,=,1,," | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output followed by a ',' |
| "xyz,=,1,\n" | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output followed by a new line |
| "xyz,=,1, . " | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output followed by a ' . ' |
| "xyz,=,1,," | All lines containing 'xyz' are split in to an array at the '=' symbol and the split[1] value will be output followed by a ',' |
| "xyz,,,\n" | All lines containing 'xyz' are output followed by a new line |
| "0,,,, \|1,,,\n" | Line 0 is written followed by a ', ' followed by line 1 folllowed by a new line |


### Value Substitution

---

All values in the json definitions are subject to 'Value Substitution'. This occurs in the GUI as well as before the exectution of an action.

The basic form of substitution replaces '%{name}' with a value defined by the given name. If name cannot be found then the value remains unchanged.

The value is located as follows:

First: The memory values are searched for a value with the given namme. 

The following will create a memory value named 'MyMemValue' with the stream of characters passed to stdout:

``` json
"stdout":"memory|MyMemValue
```

The following will substitute the value in to the first arg of a command:

```json
"args": [
    "%{MyMemValue}"
]
```

Second: The local values a searched.

If a local value is defined as follows:

```json
"commitMessage": {
    "desc": "Commit message",
    "value": "?",
    "input": true,
    "minLen": 5
}
```

The following will substitute the value in to the first arg of a command:

```json
"args": [
    "%{commitMessage}"
]
```

As the local value is an input ("input": true) it's value will be requested when the action is run. Prior to that (when the GUI is rendered) any substitutions will simply use the value in the 'value' parameter (in this case ?). This is useful for default values and for hiding actions, see 'Hide Actions' below.

Finally: The envirionment variables are searched.


```json
{
    "cmd": "cp",
    "args": ["%{HOME}/gtool-config.json", "%{HOME}/gtool-config.bak"]
}
```

The above will copy the file if the current users home directory.

Note that ALL substitution names are case sensitive.

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

---

A local value is required for encryption and decryption. The name if refered to in the 'stdin' or 'stdout' definition.

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

### Encryption (stdout)

The stdout will be written to a file and encrypted using the password (key) defined in the local value 'myPw1'.

``` json
{
    "cmd": "git",
    "args": [
        "config",
        "-l"
    ],
    "stdout": "encFile.txt",
    "outPwName": "myPw1"
}
 ```

The output from the command 'git config -l' will be writen to the 'encFile.txt' encrypted with the password defined in local value 'myPw1'.

Before the command is run a password entry dialog will be presented for entry of the password. Once entered the value is retained for all further use of the local value 'myPw1'.

### Decryption (stdin)

The system input (stdin) will be read from file 'encdFile.txt' and decrypted using the password (key) defined in the local value.

``` json
{
    "name": "Read Enc",
    "desc": "Read an encrypted file",
    "list": [
        {
            "cmd": "cat",
            "args": [],
            "stdin": "file:encdFile.txt",
            "inPwName": "myPw1"
        }
    ]
}
```

Before the command is run a password entry dialog will be presented for entry of the password. Once entered the value is retained for all further use of the local value 'myPw1'.

### Hide actions

---

The 'hide' option for Actions can use templated values to optionally hide a action.

There are many reasons to hide options:

1) The action is run when the application loads (runAtStart) of when the application terminated (runAtEnd)
2) Based on the existance of a specific file.
3) Based on the existance of a file name (or any local value) that has not been defined yet.

The way hide works:

If the value of hide = "yes" ("hide":"yes") then the action will never be displayed.

If the value of hide = starts with '%{' then the action will not be displayed.

For option 1 above use: 

```json
"hide": "yes",
```

For option 2 above: 

If a local value is defined as follows:

```json
"MyFileWatch": {
    "desc": "FileWatch: MyFile.txt",
    "value": "MyFile.txt",
    "isFileWatch": true
}
```

Then the substitution of %{MyFileWatch} will return %{MyFileWatch} if the file does NOT currently exist. 

As this substitution starts with '%{' the following 'hide' option will hide the action if the file does not exist.

```json
"hide": "%{MyFileWatch}",
```

For option 3 above: 

If a local value is defined as follows:

```json
"tempFile": {
    "desc": "Temp File",
    "input": true,
    "value": "%{}",
    "isFileName": true
}
```

Then until the file has been selected it's value will be '%{}'. This also works if the value is 'yes'.

```json
"hide": "%{tempFile}"
```

So until a file has been defined the action is hidden.
