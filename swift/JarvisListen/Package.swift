// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "JarvisListen",
    platforms: [
        .macOS(.v13)
    ],
    targets: [
        .executableTarget(
            name: "JarvisListen",
            path: "Sources"
        )
    ]
)
