# gtools

## Rational

### Run commands defined by a JSON configuration file

execute as:

``` bash
./gtool
```

The default configuration file will be config.json

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
        "cachedFields": {
            "commitMessage": {
                "desc": "Commit message",
                "default": "commit",
                "input": true
            }
        }
    },
    "actions": [
        {
            "name": "Start Code",
            "desc": "Start the dev environment",
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
| config.cachedFields | Contains a list of cached fields for substitution in args. See CachedFields below | optional |
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
"name": "Start VSCode",
"desc": "Start the dev environment",
"list": [
    {
        "cmd": "code",
        "args": [
            "."
        ],
        "in": "",
        "outFile": "",
        "outFilter": "",
        "errFile": "",
        "delay": 0,
    }
]
```

| Field name      | Description | optional |
| ----------- | ----------- | --------- |
| name | Defined the value displayed in the action button | required |
| desc | Defined the value displayed along side the action button | required |
| list | Defines a number of commands to be run one after the other | required |

### Commands (cmd):

Each command has the following fields:

| Field name      | Description | optional |
| ----------- | ----------- | --------- |
| cmd | the command to be run (excluding any arguments). E.g. ls | required |
| args | A String list of arguments. E.g. '-lta' See Agrs below | required |
| in | Input to the sysin stream if a cmd requires it. See In Filters below | optional = "" |
| outFile | Output from stdout will be written here. See Output below | optional = "" |
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

2: The result of a 'cachedFields' entry defined in the 'config' section of the config file.

```json
"cachedFields": {
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

| form      | Description |
| ----------- | ----------- |
| A_valid_file_name | The file will be deleted first, overwriting the previous content. Sysout will be written to the file. Only a single result will be written. | 
| append:A_valid_file_name | The 'append:' prefix means Sysout will be appended to the file. | 
| memory:aName | The 'memory:' prefix means Sysout will be written to the cache with the given name ('aName'). | 

### Filters

Filters can be applied to sysout (outFile) and sysin (in) as required.

Two types of filters exist In Filters and Out Filters. They are defined slightly differently. 

### Out Filters

An outFilter filters the generated sysout text. This can be written to sysout (default), a file or to cache memory. 

For example:

```json
"outFilter":"l1,s1,p1|n2,s2,p2,d2",
```

Multiple filters are devided by the '|' and each consists of up to 4 elements separated by  a ','

A single filter does not require a '|'. Multiple filters are seperated by '|'.

If the first element is a number then it is a Zero based line number. If it is not a valid integer it is treated as a string.

Any value after the 3rd comma is used as a seperator and appended to each output.

The second value is a separator used to split the line in to N parts. 

The third element is a number used to select the part (Zero based). If the seperator is defined but no part number is defined then the whole line is output. 

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

### CachedFields

---

Cached fields are persistent name value entries that can be substituted in to arguments on the fly.

These are substituted using %{fieldName} data elements.

They can be input at the start of an action or defined in the config data.

``` json
"cachedFields": {
    "commitMessage": {
        "desc": "Commit message",
        "default": "commit",
        "input": true
    }
}
```

In the above extract 'commitMessage' is the name of the field.

| Field name      | Description | optional |
| ----------- | ----------- | --------- |
| cachedFields.{name}.desc | Description used for field input | required |
| cachedFields.{name}.default | The default value for the field. This is updated if the field is input in the UI | required |
| cachedFields.{name}.input | true \| false. Input the field before running the action | optional false |
