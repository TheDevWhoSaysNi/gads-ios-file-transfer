import CryptoKit
import Foundation
import Photos

struct TransferRunner {
    private let statusName = ".gads-file-transfer.json"
    private let mediaExtensions: Set<String> = [
        "jpg", "jpeg", "png", "heic", "heif", "gif", "tif", "tiff", "bmp", "mov", "mp4", "m4v",
    ]

    func run() async -> String {
        let documents = documentsURL()
        let files = listFiles(in: documents)
        let photosOK = await ensurePhotoPermission(needsPhotos: files.contains { isMedia($0) })
        var ledger = loadLedger()
        var processed: [[String: String]] = []
        var errors: [[String: String]] = []

        for url in files {
            let name = url.lastPathComponent
            let hash = sha256(url: url)
            if isMedia(url) {
                if ledger[name] == hash {
                    processed.append(["name": name, "destination": "photos-and-files"])
                    continue
                }
                if !photosOK {
                    errors.append([
                        "name": name,
                        "destination": "files",
                        "message": "Allow GADS File Transfer to add to Photos, then send the file again.",
                    ])
                    continue
                }
                do {
                    try await saveToPhotos(url)
                    if let hash {
                        ledger[name] = hash
                    }
                    processed.append(["name": name, "destination": "photos-and-files"])
                } catch {
                    errors.append([
                        "name": name,
                        "destination": "files",
                        "message": error.localizedDescription,
                    ])
                }
            } else {
                processed.append(["name": name, "destination": "files"])
            }
        }

        saveLedger(ledger)
        writeStatus(processed: processed, errors: errors, in: documents)
        return summary(processed: processed, errors: errors)
    }

    private func ensurePhotoPermission(needsPhotos: Bool) async -> Bool {
        let current = PHPhotoLibrary.authorizationStatus(for: .addOnly)
        if current == .notDetermined {
            let next = await PHPhotoLibrary.requestAuthorization(for: .addOnly)
            return next == .authorized || next == .limited
        }
        if !needsPhotos {
            return current == .authorized || current == .limited
        }
        return current == .authorized || current == .limited
    }

    private func saveToPhotos(_ url: URL) async throws {
        try await PHPhotoLibrary.shared().performChanges {
            let request = PHAssetCreationRequest.forAsset()
            let options = PHAssetResourceCreationOptions()
            options.shouldMoveFile = false
            let type: PHAssetResourceType = isVideo(url) ? .video : .photo
            request.addResource(with: type, fileURL: url, options: options)
        }
    }

    private func listFiles(in documents: URL) -> [URL] {
        let urls = (try? FileManager.default.contentsOfDirectory(
            at: documents,
            includingPropertiesForKeys: nil,
            options: [.skipsHiddenFiles]
        )) ?? []
        return urls.filter { url in
            var isDir: ObjCBool = false
            let exists = FileManager.default.fileExists(atPath: url.path, isDirectory: &isDir)
            return exists && !isDir.boolValue && url.lastPathComponent != statusName
        }
    }

    private func isMedia(_ url: URL) -> Bool {
        mediaExtensions.contains(url.pathExtension.lowercased())
    }

    private func isVideo(_ url: URL) -> Bool {
        ["mov", "mp4", "m4v"].contains(url.pathExtension.lowercased())
    }

    private func documentsURL() -> URL {
        FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
    }

    private func supportURL() -> URL {
        let base = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
        try? FileManager.default.createDirectory(at: base, withIntermediateDirectories: true)
        return base.appendingPathComponent("import-ledger.json")
    }

    private func loadLedger() -> [String: String] {
        guard let data = try? Data(contentsOf: supportURL()) else { return [:] }
        return (try? JSONDecoder().decode([String: String].self, from: data)) ?? [:]
    }

    private func saveLedger(_ ledger: [String: String]) {
        guard let data = try? JSONEncoder().encode(ledger) else { return }
        try? data.write(to: supportURL(), options: .atomic)
    }

    private func sha256(url: URL) -> String? {
        guard let handle = try? FileHandle(forReadingFrom: url) else { return nil }
        defer { try? handle.close() }
        var hasher = SHA256()
        while true {
            let chunk = handle.readData(ofLength: 1024 * 1024)
            if chunk.isEmpty { break }
            hasher.update(data: chunk)
        }
        return hasher.finalize().map { String(format: "%02x", $0) }.joined()
    }

    private func writeStatus(processed: [[String: String]], errors: [[String: String]], in documents: URL) {
        let body: [String: Any] = [
            "state": "done",
            "processed": processed,
            "errors": errors,
        ]
        guard let data = try? JSONSerialization.data(withJSONObject: body) else { return }
        let url = documents.appendingPathComponent(statusName)
        try? data.write(to: url, options: .atomic)
    }

    private func summary(processed: [[String: String]], errors: [[String: String]]) -> String {
        if processed.isEmpty && errors.isEmpty {
            return "Nothing new. Send a file from the computer."
        }
        let photos = processed.filter { $0["destination"] == "photos-and-files" }.count
        let files = processed.count + errors.count
        if errors.isEmpty {
            if photos > 0 {
                return "Saved \(photos) into Photos. \(files) file\(files == 1 ? "" : "s") in Files."
            }
            return "\(files) file\(files == 1 ? "" : "s") in Files."
        }
        return errors[0]["message"] ?? "Some files stayed in Files only."
    }
}
