package api;

import lombok.AllArgsConstructor;
import lombok.Data;

import java.util.List;

public class Requests {

    @Data
    @AllArgsConstructor
    public static class CustomLogRequest {
        private List<Node> rows;
        private PromptScorersConfiguration promptScorersConfiguration;
    }

    @Data
    @AllArgsConstructor
    public static class CreateProjectRequest {
        private String name;
        private boolean isPublic;
        private String type;
    }

    @Data
    @AllArgsConstructor
    public static class CreateRunRequest {
        private String name;
        private String taskType;
    }

    @Data
    @AllArgsConstructor
    public static class ApiKey {
        private final String apiKey;
    }

}
