---
apiVersion: v1
kind: Secret
metadata:
  name: slack-duty-bot-${SDB_NAME}-secret
type: Opaque
data:
  token: ${SDB_SLACK_TOKEN_BASE64}
---

kind: ConfigMap
apiVersion: v1
metadata:
  name: slack-duty-bot-${SDB_NAME}-config-map
data:
  config.yaml: |-
    slack:
      keyword:
        - ${SDB_KEYWORD}
      group:
        id: ${SDB_SLACK_GROUP_ID}
        name: ${SDB_SLACK_GROUP_NAME}
    duties:
      - [${SDB_SLACK_DEFAULT_USER}]
      - [${SDB_SLACK_DEFAULT_USER}]
      - [${SDB_SLACK_DEFAULT_USER}]
      - [${SDB_SLACK_DEFAULT_USER}]
      - [${SDB_SLACK_DEFAULT_USER}]
      - [${SDB_SLACK_DEFAULT_USER}]
      - [${SDB_SLACK_DEFAULT_USER}]
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: slack-duty-bot-${SDB_NAME}
  labels:
    app: slack-duty-bot-${SDB_NAME}
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 50%
      maxSurge: 1
  template:
    metadata:
      labels:
        app: slack-duty-bot-${SDB_NAME}
    spec:
      containers:
      - name: slack-duty-bot-${SDB_NAME}
        image: iqoption/slack-duty-bot:${SDB_TAG}
        imagePullPolicy: Always
        args: ["--config.path=/etc/slack-duty-bot"]
        env:
        - name: SDB_SLACK_TOKEN
          valueFrom:
            secretKeyRef:
              name: slack-duty-bot-${SDB_NAME}-secret
              key: token
        volumeMounts:
        - name: slack-duty-bot-${SDB_NAME}-config-volume
          mountPath: /etc/slack-duty-bot
      volumes:
        - name: slack-duty-bot-${SDB_NAME}-config-volume
          configMap:
            name: slack-duty-bot-${SDB_NAME}-config-map
      restartPolicy: Always
      terminationGracePeriodSeconds: 30
