podTemplate(yaml: '''
apiVersion: v1
kind: Pod
spec:
  containers:
    - name: monaco-build
      image: golang
      imagePullPolicy: Always
      command:
      - cat
      tty: true
      resources:
        requests:
          cpu: "100m"
          memory: "100Mi"
        limits:
          cpu: "2"
          memory: "16Gi"
  securityContext:
    runAsUser: 1000
'''
) {
    node(POD_LABEL) {
        stage("HELLO") {
            echo "hello world"
        }
    }
}
