<?php

require 'vendor/autoload.php';

$apiKey = "YOUR_API_KEY";
$baseUrl = "YOUR_GALILEO_BASE_URL";

function generateUuid4() {
    return sprintf('%04x%04x-%04x-%04x-%04x-%04x%04x%04x',
        mt_rand(0, 0xffff), mt_rand(0, 0xffff),
        mt_rand(0, 0xffff),
        mt_rand(0, 0x0fff) | 0x4000,
        mt_rand(0, 0x3fff) | 0x8000,
        mt_rand(0, 0xffff), mt_rand(0, 0xffff), mt_rand(0, 0xffff)
    );
}

$client = new GuzzleHttp\Client([
    'base_uri' => $baseUrl,
    'headers' => [
        'Galileo-API-Key' => $apiKey,
        'Content-Type' => 'application/json'
    ]
]);

try {
    $projectName = 'galileo_php_scripts__Apr-13-2025-23-56-03'; // Default project name

    if (in_array('--new', $argv)) {
        echo "Creating new project...\n";
        $projectName = 'galileo_php_scripts__' . date('M-d-Y-H-i-s');
        $projectData = [
            'name' => $projectName,
            'created_by' => generateUuid4(),
            'type' => 'prompt_evaluation',
            'create_example_templates' => false
        ];
        
        $projectResponse = $client->post('/projects', [
            'json' => $projectData
        ]);

        $projectResult = json_decode($projectResponse->getBody(), true);
        echo "Created project with project ID: " . $projectResult['id'] . "\n\n";
    } elseif (in_array('--existing', $argv)) {
        $existingIndex = array_search('--existing', $argv);
        if (isset($argv[$existingIndex + 1])) {
            $projectName = $argv[$existingIndex + 1];
            echo "Using existing project: " . $projectName . "\n";
        } else {
            throw new Exception("Please provide a project name after --existing flag");
        }
    }

    echo "Creating evaluate run...\n";
    $runData = [
        'project_name' => $projectName,
        'run_name' => 'evaluate_run_' . date('Ymd_His'),
        'scorers' => [
            [
                'name' => 'correctness'
            ],
            [
                'name' => 'output_pii'
            ]
        ],
        'workflows' => [
            [
                'created_at_ns' => time() * 1000000000, // Current time in nanoseconds
                'duration_ns' => 0,
                'input' => 'who is a smart LLM?',
                'metadata' => new stdClass(),
                'name' => 'llm',
                'output' => 'I am!',
                'type' => 'llm'
            ]
        ]
    ];

    $runResponse = $client->post('/evaluate/runs', [
        'json' => $runData
    ]);

    $runResult = json_decode($runResponse->getBody(), true);
    echo "Evaluate run created successfully!\n";
    echo "Project ID: " . $runResult['project_id'] . "\n";
    echo "Run ID: " . $runResult['run_id'] . "\n";
    echo "Workflows count: " . $runResult['workflows_count'] . "\n";
    echo "Records count: " . $runResult['records_count'] . "\n";

} catch (Exception $e) {
    echo "Error: " . $e->getMessage() . "\n";
    if ($e instanceof GuzzleHttp\Exception\RequestException && $e->hasResponse()) {
        $response = $e->getResponse();
        echo "Response status code: " . $response->getStatusCode() . "\n";
        echo "Response body: " . $response->getBody() . "\n";
    }
} 
