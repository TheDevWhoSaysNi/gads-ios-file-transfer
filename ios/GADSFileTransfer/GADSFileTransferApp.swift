import SwiftUI

@main
struct GADSFileTransferApp: App {
    var body: some Scene {
        WindowGroup {
            TransferView()
        }
    }
}

struct TransferView: View {
    @Environment(\.scenePhase) private var scenePhase
    @State private var line = "Waiting for a file."

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("GADS File Transfer")
                .font(.title2.weight(.semibold))
            Text("Photos and videos are copied into Photos. Every file stays in Files, in this app’s folder.")
                .foregroundStyle(.secondary)
            Text(line)
                .font(.callout)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .padding(24)
        .task(id: scenePhase) {
            guard scenePhase == .active else { return }
            line = "Checking for new files…"
            let summary = await Task.detached(priority: .userInitiated) {
                await TransferRunner().run()
            }.value
            line = summary
        }
    }
}
