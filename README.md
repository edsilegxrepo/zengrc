# ZenGRC Attachment Downloader

## 1. Application Overview and Objectives

The ZenGRC Attachment Downloader is a command-line application written in Go that downloads attachments and all associated metadata from the ZenGRC API.

The primary objectives of this application are:
- To provide a reliable way to back up or archive ZenGRC records and their attachments.
- To process records concurrently to maximize download speed and efficiency.
- To offer a secure and configurable way to interact with the ZenGRC API.

For each record found, the application creates a dedicated folder, saves the complete record metadata as a `metadata.json` file, and downloads all associated attachments into that folder.

## 2. Architecture and Design Choices

The application is designed with a focus on performance, security, and maintainability.

- **Modularity:** The codebase is split into two main files:
    - `main.go`: Contains the application's entry point, command-line flag parsing, and the concurrency logic (worker pool).
    - `client.go`: Contains a dedicated API client for all interactions with the ZenGRC API, separating the application logic from the API communication logic.

- **Concurrency:** The application uses a worker pool pattern to process records concurrently. This allows for multiple records to be downloaded at the same time, significantly improving performance when dealing with a large number of records. Errors from concurrent workers are collected in a dedicated channel and reported at the end of the execution, ensuring that no failure goes unnoticed.

- **Security:**
    - **Secure File Permissions:** To prevent unauthorized access in multi-user environments, directories are created with `0750` permissions and metadata files with `0600`.
    - **No Hardcoded Credentials:** The API token is passed via a command-line flag, preventing sensitive information from being stored in the source code.
    - **File Overwrite Protection:** By default, the application will not overwrite existing files, preventing accidental data loss. This can be overridden with the `-overwrite` flag.
    - **Static Analysis Compliance:** The codebase is regularly audited with `gosec` to ensure security best practices are followed.

- **Performance:** The HTTP client is configured with a custom transport to optimize connection pooling and reuse, which is crucial for an application that makes a large number of API calls.

## 3. Attachment Management

Attachments are associated with their corresponding metadata in two ways:

1.  **By Folder Structure:** The application creates a dedicated folder for each record, named `record_<ID>`, where `<ID>` is the unique identifier of the record. The `metadata.json` file and all associated attachments for that specific record are placed inside this folder. This provides a clear and organized grouping of a record's metadata and its corresponding files.

2.  **Programmatically via API Calls:** The application's logic ensures this association:
    *   First, it fetches a list of all `Request` records.
    *   Then, for each individual `Request` record (e.g., the one with `ID=123`), it makes a separate API call to an endpoint like `/api/v2/requests/123/attachments`. This endpoint specifically returns a list of all attachments that belong *only* to that record.
    *   Finally, it downloads those attachments into the `record_123` folder, alongside the `metadata.json` for that same record.

This ensures that the association is guaranteed by both the API's design and the application's workflow.

## 4. Metadata Details

The `metadata.json` file saved for each record contains the following fields, extracted directly from the ZenGRC API:

| Field                | Data Type                      | Description                                                  |
|----------------------|--------------------------------|--------------------------------------------------------------|
| `id`                 | integer                        | The unique identifier for the request record.                |
| `title`              | string                         | The title of the request.                                    |
| `code`               | string                         | The code or reference number for the request.                |
| `assignees`          | array of `PersonInfo` objects  | The users assigned to the request.                           |
| `audit`              | `AuditInfo` object             | Information about the audit associated with the request.     |
| `created_at`         | string (date-time)             | The timestamp when the request was created.                  |
| `custom_attributes`  | map of `CustomAttrValue` objects | A map of custom attributes associated with the request.      |
| `description`        | string (nullable)              | The description of the request.                              |
| `due_date`           | string (date, nullable)        | The due date for the request.                                |
| `links`              | `DetailsLinks` object          | Links related to the request, including a self-referencing URL. |
| `mapped`             | `RequestMapped` object         | Information about objects mapped to the request.             |
| `notes`              | string (nullable)              | Any notes associated with the request.                       |
| `notify_assignee`    | boolean (nullable)             | A flag indicating if assignees should be notified.           |
| `requesters`         | array of `PersonInfo` objects  | The users who created the request.                           |
| `reviewers`          | array of `ReviewerStatus` objects | The users responsible for reviewing the request.             |
| `start_date`         | string (date)                  | The start date of the request.                               |
| `status`             | string                         | The current status of the request (e.g., "Open", "Completed"). |
| `tags`               | array of strings               | Tags associated with the request.                            |
| `test`               | string (nullable)              | The test plan or procedure for the request.                  |
| `type`               | string                         | The object type, which is always "Request".                  |
| `updated_at`         | string (date-time)             | The timestamp when the request was last updated.             |
| `verifiers`          | array of `PersonInfo` objects  | The users responsible for verifying the request.             |

## 5. Command-Line Arguments

The application is configured using the following command-line flags:

| Flag          | Type    | Default                | Description                                                              |
|---------------|---------|------------------------|--------------------------------------------------------------------------|
| `-api-url`    | string  | (none)                 | **(Required)** The URL of your ZenGRC API instance (e.g., `https://acme.api.zengrc.com`). |
| `-token`      | string  | (none)                 | **(Required)** Your ZenGRC API authentication token in the format `key_id:key_secret`. |
| `-output-dir` | string  | `./zengrc_attachments` | The directory where the attachments and metadata will be saved.            |
| `-workers`    | int     | `5`                    | The number of concurrent workers to use for downloading.                 |
| `-overwrite`  | bool    | `false`                | If set to `true`, the application will overwrite existing files.         |
| `-version`    | bool    | `false`                | Print the application version and exit.                                  |

## 6. Examples

### Basic Usage

The following command will download all records and their attachments to the default `./zengrc_attachments` directory, using 5 concurrent workers.

```bash
./zengrc \
  -api-url "https://your-instance.api.zengrc.com" \
  -token "your_key_id:your_key_secret"
```

### Custom Output Directory and Worker Count

This example downloads the records to a custom directory (`/path/to/my/backup`) and uses 10 concurrent workers.

```bash
./zengrc \
  -api-url "https://your-instance.api.zengrc.com" \
  -token "your_key_id:your_key_secret" \
  -output-dir "/path/to/my/backup" \
  -workers 10
```

### Overwriting Existing Files

If you need to re-download all files and overwrite any that already exist, use the `-overwrite` flag.

```bash
./zengrc \
  -api-url "https://your-instance.api.zengrc.com" \
  -token "your_key_id:your_key_secret" \
  -overwrite
```