#!/usr/bin/groovy
@Library('github.com/fabric8io/fabric8-pipeline-library@master')
def dummy
goNode{
  dockerNode{
    def v = goRelease{
      githubOrganisation = 'funktionio'
      dockerOrganisation = 'funktion'
      project = 'funktion'
    }
  }
}

def updateDownstreamDependencies(v) {
  pushPomPropertyChangePR {
    propertyName = 'funktion.version'
    projects = [
            'fabric8io/fabric8-platform'
    ]
    version = v
  }
}
