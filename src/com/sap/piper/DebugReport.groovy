package com.sap.piper

import com.cloudbees.groovy.cps.NonCPS
import groovy.text.SimpleTemplateEngine

@Singleton
class DebugReport {
    String fileName
    String projectIdentifier = null
    Map environment = ['environment': 'custom']
    String buildTool = null
    Map modulesMap = [:]
    List npmModules = []
    Set plugins = []
    Map gitRepo = [:]
    Map localExtensions = [:]
    String globalExtensionRepository = null
    Map globalExtensions = [:]
    String globalExtensionConfigurationFilePath = null
    String sharedConfigFilePath = null
    Set additionalSharedLibraries = []
    Map failedBuild = [:]
    boolean shareConfidentialInformation

    /**
     * Initialize debug report information from the environment variables.
     *
     * @param env The Jenkins global 'env' variable.
     */
    void initFromEnvironment(def env) {
        Set buildDetails = []
        buildDetails.add('Jenkins Version | ' + env.JENKINS_VERSION)
        buildDetails.add('JAVA Version | ' + env.JAVA_VERSION)
        environment.put('build_details', buildDetails)

        if (!Boolean.valueOf(env.ON_K8S) && EnvironmentUtils.cxServerDirectoryExists()) {
            environment.put('environment', 'cx-server')

            String serverConfigContents = getServerConfigContents(
                '/var/cx-server/server.cfg',
                '/workspace/var/cx-server/server.cfg')
            String docker_image = EnvironmentUtils.getDockerFile(serverConfigContents)
            environment.put('docker_image', docker_image)
        }
    }

    private static String getServerConfigContents(String... possibleFileLocations) {
        for (String location in possibleFileLocations) {
            File file = new File(location)
            if (file.exists())
                return file.getText('UTF-8')
        }
        return ''
    }

    /**
     * Pulls and stores repository information from the provided Map for later inclusion in the debug report.
     *
     * @param scmCheckoutResult A Map including information about the checked out project,
     * i.e. as returned by the Jenkins checkout() function.
     */
    void setGitRepoInfo(Map scmCheckoutResult) {
        if (!scmCheckoutResult.GIT_URL)
            return

        gitRepo.put('URI', scmCheckoutResult.GIT_URL)
        if (scmCheckoutResult.GIT_LOCAL_BRANCH) {
            gitRepo.put('branch', scmCheckoutResult.GIT_LOCAL_BRANCH)
        } else {
            gitRepo.put('branch', scmCheckoutResult.GIT_BRANCH)
        }
    }

    /**
     * Stores crash information for a failed step. Multiple calls to this method overwrite already
     * stored information, only the information stored last will appear in the debug report. In the
     * current use-case where this can be called multiple times, all 'unstable' steps are listed in
     * the 'unstableSteps' entry of the commonPipelineEnvironment.
     *
     * @param stepName      The name of the crashed step or stage
     * @param err           The Throwable that was thrown
     * @param isResilientAndNotMandatory
     *                      Whether the step configuration indicated that this as
     *                      "resilient" step which is not included in the list of mandatory steps.
     */
    void storeStepFailure(String stepName, Throwable err, boolean isResilientAndNotMandatory) {
        failedBuild.put('stage', stepName)
        failedBuild.put('reason', err)
        failedBuild.put('stack_trace', err.getStackTrace())
        if (isResilientAndNotMandatory) {
            failedBuild.put('isResilient', 'true')
        } else {
            failedBuild.remove('isResilient')
        }
    }

    String generateReport(Script script) {
        String template = script.libraryResource 'debug_report.txt'

        if (!projectIdentifier) {
            projectIdentifier = 'NOT_SET'
        }

        try {
            Jenkins.instance.getPluginManager().getPlugins().each {
                plugins.add("${it.getShortName()} | ${it.getVersion()} | ${it.getDisplayName()}")
            }
        } catch (Throwable t) {
            // Ignore
        }

        Map binding = getProperties()
        Date now = new Date()

        binding.utcTimestamp = now.format('yyyy-MM-dd HH:mm', TimeZone.getTimeZone('UTC'))
        String fileNameTimestamp = now.format('yyyy-MM-dd-HH-mm', TimeZone.getTimeZone('UTC'))

        if (shareConfidentialInformation) {
            fileName = "confidential_debug_log_${fileNameTimestamp}_${projectIdentifier}.txt"
        } else {
            fileName = "redacted_debug_log_${fileNameTimestamp}_${projectIdentifier}.txt"
        }

        return fillTemplate(template, binding)
    }

    @NonCPS
    private String fillTemplate(String template, binding) {
        def engine = new SimpleTemplateEngine()
        return engine.createTemplate(template).make(binding)
    }
}
