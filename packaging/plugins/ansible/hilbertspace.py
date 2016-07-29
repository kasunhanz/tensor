# Make coding more python3-ish
from __future__ import (absolute_import, division, print_function)

__metaclass__ = type

import os
import yaml
from pymongo import MongoClient

from ansible.plugins.callback import CallbackBase


class CallbackModule(CallbackBase):
    """
    required "sudo pip install pymongo"

    logs ansible-playbook and ansible runs to a syslog server in json format
    make sure you have in ansible.cfg:
        callback_plugins   = <path_to_callback_plugins_folder>
    and put the plugin in <path_to_callback_plugins_folder>

    This plugin makes use of the following environment variables:
        HS_TASK_ID   (must)
        HS_TASK_TYPE     (optional): defaults to 1
    """
    CALLBACK_VERSION = 2.0
    CALLBACK_TYPE = 'aggregate'
    CALLBACK_NAME = 'hilbertspace'
    CALLBACK_NEEDS_WHITELIST = True

    def __init__(self):

        super(CallbackModule, self).__init__()

        # Get configuration object from hilbertspace
        # configuration file
        with open("/etc/hilbertspace.conf", 'r') as stream:
            try:
                self.config = yaml.load(stream)
            except yaml.YAMLError as exc:
                self.display.warning('Could not parse configuration file')

        # Build connection string and connect
        connection_str = "mongodb://{}/".format(",".join(self.config['mongodb']['hosts']))

        if len(self.config['mongodb']['replica_set']):
            connection_str += "?replicaSet={}".format(self.config['mongodb']['replica_set'])

        mongoclient = MongoClient(connection_str)

        # Get database
        self.db = self.mongoclient[self.config['mongodb']['name']]
        # authentication using configuration file
        self.db.authenticate(self.config['mongodb']['user'], self.config['mongodb']['pass'])

        self.taskID = os.getenv('HS_TASK_ID')
        self.taskType = os.getenv('HS_TASK_TYPE', 1)

    def runner_on_failed(self, host, res, ignore_errors=False):
        result = self.db.addhoc_tasks.update_one(
            {"_id": self.taskID},
            {
                "$push": {
                    "task_status": {
                        "execution": "FAILED",
                        "host": host,
                        "message": res
                    }
                }
            }
        )

    def runner_on_ok(self, host, res):
        result = self.db.addhoc_tasks.update_one(
            {"_id": self.taskID},
            {
                "$push": {
                    "task_status": {
                        "execution": "SUCCESS",
                        "host": host,
                        "message": res
                    }
                }
            }
        )

    def runner_on_skipped(self, host, item=None):
        result = self.db.addhoc_tasks.update_one(
            {"_id": self.taskID},
            {
                "$push": {
                    "task_status": {
                        "execution": "SKIPPED",
                        "host": host
                    }
                }
            }
        )

    def runner_on_unreachable(self, host, res):
        result = self.db.addhoc_tasks.update_one(
            {"_id": self.taskID},
            {
                "$push": {
                    "task_status": {
                        "execution": "UNREACHABLE",
                        "host": host,
                        "message": res
                    }
                }
            }
        )

    def runner_on_async_failed(self, host, res):
        result = self.db.addhoc_tasks.update_one(
            {"_id": self.taskID},
            {
                "$push": {
                    "task_status": {
                        "execution": "ASYNC_FAILED",
                        "host": host,
                        "message": res
                    }
                }
            }
        )
