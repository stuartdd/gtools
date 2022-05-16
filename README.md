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

Each action is defined as follows:

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

### Output

### In Filters

### Out Filters

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
