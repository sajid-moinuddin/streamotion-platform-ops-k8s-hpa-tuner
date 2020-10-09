pipeline {
  agent {
    label "jenkins-go"
//    label "streamotion-maven"
  }

  environment {
    ORG = 'fsa-streamotion'
    APP_NAME = 'streamotion-platform-ops-k8s-hpa-tuner'
  }

  stages {
    stage('Test') {
      when {
        branch 'PR-*'
      }
      steps {
        container('streamotion-go') {
//        container('maven') {
//          TODO this does not work due to docker in docker running in jenkins - leaving it for later
//          sh "make kind-test-setup"
//          sh "make kind-tests"
        }
      }
    }


    stage('Push To ECR') {
      when {
//        branch 'master'
        branch 'PR-*'
      }
      steps {
        container('streamotion-go') {
//        container('maven') {

          // ensure we're not on a detached head
          sh "git config --global credential.helper store"
          sh "jx step git credentials"

          sh "echo \$(jx-release-version) > VERSION"
          sh "jx step tag --version \$(cat VERSION)"
          sh "skaffold version"
          sh "export VERSION=`cat VERSION` && skaffold build -f skaffold.yaml"
          sh "export VERSION=latest && skaffold build -f skaffold.yaml"
          script {
            def buildVersion = readFile "${env.WORKSPACE}/VERSION"
            currentBuild.description = "${DOCKER_REGISTRY}/streamotion-platform-ops-k8s-hpa-tuner:$buildVersion"
            currentBuild.displayName = "$buildVersion"
          }
        }
      }
    }

    stage('Promote to Environments') {
      when {
//        branch 'master'
        branch 'PR-*'
      }
      steps {
        container('streamotion-go') {
//        container('maven') {
          sh "mv charts/helm-release  charts/$APP_NAME"
          dir("charts/$APP_NAME") {
            sh "jx step changelog --generate-yaml=false --version v\$(cat ../../VERSION)"

            sh "make release"
            // promote through all 'Staging' promotion Environments
            sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=platform-bedrock --version \$(cat ../../VERSION)"
            //          sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=commerce-staging --version \$(cat ../../VERSION)"
            //            sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=content-staging --version \$(cat ../../VERSION)"
            //            sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=streamtech-staging --version \$(cat ../../VERSION)"

            // promote through all 'Production' promotion Environments
            //          sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=commerce-production --version \$(cat ../../VERSION)"
            // sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=content-production --version \$(cat ../../VERSION)"
            // sh "jx promote -b --no-poll=true  --helm-repo-url=$CHART_REPOSITORY --no-poll=true --no-merge=true --no-wait=true --env=streamtech-production --version \$(cat ../../VERSION)"

          }
        }
      }
    }
  }
  post {
    always {
      cleanWs()
    }
  }
}
