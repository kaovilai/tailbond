// swift-tools-version:5.9
// tailbond's menu bar app for macOS (docs/macos.md, "Menu bar"): a
// MenuBarExtra over the local tailbondd's API. No paid Apple account: the
// app is ad-hoc signed by packaging/macos/build-app.sh.
import PackageDescription

let package = Package(
    name: "TailbondMenu",
    platforms: [.macOS(.v13)],
    products: [
        .executable(name: "tailbond", targets: ["TailbondMenu"]),
        .library(name: "TailbondMenuKit", targets: ["TailbondMenuKit"]),
    ],
    targets: [
        .target(name: "TailbondMenuKit"),
        .executableTarget(name: "TailbondMenu", dependencies: ["TailbondMenuKit"]),
        .testTarget(
            name: "TailbondMenuKitTests",
            dependencies: ["TailbondMenuKit"],
            resources: [.copy("Fixtures")]
        ),
    ]
)
