package api;

import lombok.AllArgsConstructor;
import lombok.Data;

import java.util.List;

public class Responses {

    @Data
    @AllArgsConstructor
    public static class CreateProjectResponse {
        private String name;
        private String createdBy;
        private boolean isPublic;
        private String type;
        private String id;
        private String createdAt;
        private String updatedAt;
    }

    @Data
    @AllArgsConstructor
    public class CreateRunResponse {
        private String name;
        private String projectId;
        private String createdBy;
        private int numSamples;
        private boolean winner;
        private String datasetHash;
        private String id;
        private String createdAt;
        private String updatedAt;
        private int taskType;
        private String lastUpdatedBy;
        private List<RunTag> runTags;

        @Data
        public class RunTag {
            private String key;
            private String value;
            private String tagType;
            private String projectId;
            private String runId;
            private String createdBy;
            private String id;
            private String createdAt;
            private String updatedAt;
        }
    }

    @Data
    @AllArgsConstructor
    public static class LoginResponse {
        private String accessToken;
        private String tokenType;
    }

}
