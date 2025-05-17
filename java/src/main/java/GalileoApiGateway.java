import api.Node;
import api.NodeBuilder;
import api.PromptScorersConfiguration;
import api.Requests;
import api.Responses;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.reflect.TypeToken;
import utils.RandomNameGenerator;
import utils.SnakeCaseNamingStrategy;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.Collections;
import java.util.UUID;

public class GalileoApiGateway {

    private static String ROOT_URL;
    private static String GALILEO_API_KEY;

    public Responses.LoginResponse login() {
        String loginURL = ROOT_URL + "/login/api_key";
        HttpURLConnection connection = null;
        BufferedReader reader = null;

        try {
            URL url = new URL(loginURL);
            connection = (HttpURLConnection) url.openConnection();
            connection.setRequestMethod("POST");
            connection.setRequestProperty("Content-Type", "application/json");
            connection.setDoOutput(true);

            String jsonInputString = getGson().toJson(new Requests.ApiKey(GALILEO_API_KEY));

            try (OutputStream os = connection.getOutputStream()) {
                byte[] input = jsonInputString.getBytes(StandardCharsets.UTF_8);
                os.write(input, 0, input.length);
            }

            int responseCode = connection.getResponseCode();

            if (responseCode == HttpURLConnection.HTTP_OK) {
                reader = new BufferedReader(new InputStreamReader(connection.getInputStream(), StandardCharsets.UTF_8));
                StringBuilder response = new StringBuilder();
                String line;
                while ((line = reader.readLine()) != null) {
                    response.append(line.trim());
                }
                System.out.println("Login Response: " + response);
                return getGson().fromJson(response.toString(), new TypeToken<Responses.LoginResponse>() {
                }.getType());
            } else {
                System.err.println("POST request failed. Response Code: " + responseCode);
                readErrorResponse(connection);
                return null;
            }
        } catch (Exception e) {
            e.printStackTrace();
        } finally {
            closeResources(reader, connection);
        }
        return null;
    }

    public Responses.CreateProjectResponse createProject(String authToken) {
        String projectsURL = ROOT_URL + "/projects";
        HttpURLConnection connection = null;

        try {
            URL url = new URL(projectsURL);
            connection = (HttpURLConnection) url.openConnection();
            connection.setRequestMethod("POST");
            connection.setRequestProperty("Content-Type", "application/json");
            connection.setRequestProperty("Authorization", "Bearer " + authToken);
            connection.setDoOutput(true);

            String createProjectJson = getGson().toJson(new Requests.CreateProjectRequest(
                    "project__" + UUID.randomUUID(), false, "prompt_evaluation")
            );

            try (OutputStream os = connection.getOutputStream()) {
                byte[] input = createProjectJson.getBytes(StandardCharsets.UTF_8);
                os.write(input, 0, input.length);
            }

            int responseCode = connection.getResponseCode();
            if (responseCode == HttpURLConnection.HTTP_OK) {
                return parseResponse(connection, Responses.CreateProjectResponse.class);
            } else {
                handleResponseError(responseCode);
                return null;
            }
        } catch (Exception e) {
            e.printStackTrace();
        } finally {
            if (connection != null) {
                connection.disconnect();
            }
        }
        return null;
    }

    public Responses.CreateRunResponse createRun(String authToken, String projectId, String runName) {
        String createRunURL = ROOT_URL + "/projects/" + projectId + "/runs";
        HttpURLConnection connection = null;

        try {
            URL url = new URL(createRunURL);
            connection = (HttpURLConnection) url.openConnection();
            connection.setRequestMethod("POST");
            connection.setRequestProperty("Content-Type", "application/json");
            connection.setRequestProperty("Authorization", "Bearer " + authToken);
            connection.setDoOutput(true);

            String createRunJson = getGson().toJson(new Requests.CreateRunRequest(runName, "prompt_chain"));

            try (OutputStream os = connection.getOutputStream()) {
                byte[] input = createRunJson.getBytes(StandardCharsets.UTF_8);
                os.write(input, 0, input.length);
            }

            int responseCode = connection.getResponseCode();
            if (responseCode == HttpURLConnection.HTTP_OK) {
                return parseResponse(connection, Responses.CreateRunResponse.class);
            } else {
                handleResponseError(responseCode);
                return null;
            }
        } catch (Exception e) {
            e.printStackTrace();
        } finally {
            if (connection != null) {
                connection.disconnect();
            }
        }
        return null;
    }

    public void customLog(String authToken, String projectId, String runId) {
        String uuid = UUID.randomUUID().toString();
        Node node = new NodeBuilder()
                .setNodeId(uuid)
                .setNodeType("llm")
                .setNodeName("LLM")
                .setNodeInput("Tell me a joke about bears!")
                .setNodeOutput("Here is one: Why did the bear go to the doctor? Because it had a grizzly cough!")
                .setChainRootId(uuid)
                .setStep(0)
                .setHasChildren(false)
                .setLatency(0L)
                .build();

        // Choose metrics
        PromptScorersConfiguration psc =
                PromptScorersConfiguration.builder()
                        .factuality(true)
                        .groundedness(true)
                        .build();

        String customLogURL = ROOT_URL + "/projects/" + projectId + "/runs/" + runId + "/chains/ingest";

        HttpURLConnection connection = null;

        try {
            URL url = new URL(customLogURL);
            connection = (HttpURLConnection) url.openConnection();
            connection.setRequestMethod("POST");
            connection.setRequestProperty("Content-Type", "application/json");
            connection.setRequestProperty("Authorization", "Bearer " + authToken);
            connection.setDoOutput(true);

            String jsonInputString = getGson().toJson(new Requests.CustomLogRequest(Collections.singletonList(node), psc));

            try (OutputStream os = connection.getOutputStream()) {
                byte[] input = jsonInputString.getBytes(StandardCharsets.UTF_8);
                os.write(input, 0, input.length);
            }

            int responseCode = connection.getResponseCode();
            debugResponse(connection, responseCode, jsonInputString);
        } catch (Exception e) {
            e.printStackTrace();
        } finally {
            if (connection != null) {
                connection.disconnect();
            }
        }
    }

    private static Gson getGson() {
        return new GsonBuilder().setFieldNamingStrategy(new SnakeCaseNamingStrategy()).create();
    }

    private static <T> T parseResponse(HttpURLConnection connection, Class<T> clazz) throws IOException {
        try (BufferedReader in = new BufferedReader(new InputStreamReader(connection.getInputStream(), StandardCharsets.UTF_8))) {
            StringBuilder response = new StringBuilder();
            String line;
            while ((line = in.readLine()) != null) {
                response.append(line.trim());
            }
            return getGson().fromJson(response.toString(), clazz);
        }
    }

    private static void handleResponseError(int responseCode) {
        if (responseCode == HttpURLConnection.HTTP_UNAUTHORIZED) {
            System.err.println("POST request Unauthorized. Code: " + responseCode);
        } else {
            System.err.println("Other error. Code: " + responseCode);
        }
    }

    private static void debugResponse(HttpURLConnection connection, int responseCode, String jsonInputString) throws IOException {
        if (responseCode == HttpURLConnection.HTTP_OK) {
            readSuccessResponse(connection);
        } else {
            System.err.println("Error in Response. Status:" + responseCode);
            System.err.println("Request Payload: " + jsonInputString);
            readErrorResponse(connection);
        }
    }

    private static void readSuccessResponse(HttpURLConnection connection) throws IOException {
        try (BufferedReader br = new BufferedReader(new InputStreamReader(connection.getInputStream(), StandardCharsets.UTF_8))) {
            StringBuilder response = new StringBuilder();
            String responseLine;
            while ((responseLine = br.readLine()) != null) {
                response.append(responseLine.trim());
            }
            System.out.println("Success Response: " + response);
        }
    }

    private static void readErrorResponse(HttpURLConnection connection) throws IOException {
        try (BufferedReader br = new BufferedReader(new InputStreamReader(connection.getErrorStream(), StandardCharsets.UTF_8))) {
            StringBuilder response = new StringBuilder();
            String responseLine;
            while ((responseLine = br.readLine()) != null) {
                response.append(responseLine.trim());
            }
            System.err.println("Error Response: " + response);
        }
    }

    private static void closeResources(BufferedReader reader, HttpURLConnection connection) {
        try {
            if (reader != null) {
                reader.close();
            }
        } catch (IOException e) {
            e.printStackTrace();
        }
        if (connection != null) {
            connection.disconnect();
        }
    }

    public static void main(String[] args) {
        if (args.length != 2) {
            System.err.println("Pass in API KEY and URL as arguments");
            System.exit(1);
        }
        GALILEO_API_KEY = args[0];
        ROOT_URL = args[1];
        GalileoApiGateway apiGateway = new GalileoApiGateway();

        // Login
        System.out.println("=== LOGGING IN ===");
        Responses.LoginResponse loginResponse = apiGateway.login();

        // Create project
        if (loginResponse != null) {
            System.out.println("=== CREATING PROJECT ===");
            Responses.CreateProjectResponse createProjectResponse = apiGateway.createProject(loginResponse.getAccessToken());
            if (createProjectResponse != null) {
                System.out.println("PROJECT CREATED: " + createProjectResponse.getName());

                // Create run
                String runName = RandomNameGenerator.generateRandomName();
                System.out.println("=== CREATING RUN ===");
                Responses.CreateRunResponse runResponse = apiGateway.createRun(loginResponse.getAccessToken(), createProjectResponse.getId(), runName);

                // Custom Log
                if (runResponse != null) {
                    System.out.println("RUN CREATED: " + runResponse.getName());
                    System.out.println("=== LOGGING DATA TO GALILEO ===");
                    apiGateway.customLog(loginResponse.getAccessToken(), createProjectResponse.getId(), runResponse.getId());
                }
            }
        }
    }
}
