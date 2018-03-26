{
    "version": 0.1,
    "tabs": [
        {
            "title": "IBM MQ Container",
            "numColumns": 2,
            "model": {
                "title": "",
                "rows": [
                    {
                        "columns": [
                            {
                                "widgets": [
                                    {
                                        "type": "channel",
                                        "config": {
                                            "selectedQM": "{{ .QueueManagerName }}",
                                            "showSysObjs": false,
                                            "sizex": 1,
                                            "sizey": 1,
                                            "subType": "all"
                                        },
                                        "title": "Channels on {{ .QueueManagerName }}",
                                        "titleTemplateUrl": "adf/templates/widget-title.html",
                                        "gridsterrow": 0,
                                        "gridstercol": 1
                                    },
                                    {
                                        "type": "topic",
                                        "config": {
                                            "selectedQM": "{{ .QueueManagerName }}",
                                            "showSysObjs": false,
                                            "sizex": 1,
                                            "sizey": 1
                                        },
                                        "title": "Topics on {{ .QueueManagerName }}",
                                        "titleTemplateUrl": "adf/templates/widget-title.html",
                                        "gridsterrow": 1,
                                        "gridstercol": 1
                                    },
                                    {
                                        "type": "queue",
                                        "config": {
                                            "selectedQM": "{{ .QueueManagerName }}",
                                            "showSysObjs": false,
                                            "sizex": 1,
                                            "sizey": 1,
                                            "subType": "all"
                                        },
                                        "title": "Queues on {{ .QueueManagerName }}",
                                        "titleTemplateUrl": "adf/templates/widget-title.html",
                                        "gridsterrow": 1,
                                        "gridstercol": 0
                                    },
                                    {
                                        "type": "queuemanager",
                                        "gridstercol": 0,
                                        "gridsterrow": 0,
                                        "config": {
                                            "type": "local",
                                            "sizex": 1,
                                            "sizey": 1,
                                            "customTitle": "Queue Manager"
                                        },
                                        "title": "Queue Manager",
                                        "titleTemplateUrl": "adf/templates/widget-title.html"
                                    }
                                ]
                            }
                        ]
                    }
                ],
                "titleTemplateUrl": "adf/templates/dashboard-title.html"
            },
            "isMobile": false
        }
    ]
}