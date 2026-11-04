// swift-tools-version:5.9
// tailbond Share: the Pair & Share relay for iPhone (docs/ios.md).
import PackageDescription

let package = Package(
    name: "TailbondShareKit",
    platforms: [.iOS(.v16), .macOS(.v13)],
    products: [
        .library(name: "TailbondShareKit", targets: ["TailbondShareKit"]),
    ],
    targets: [
        .target(name: "TailbondShareKit"),
        .testTarget(
            name: "TailbondShareKitTests",
            dependencies: ["TailbondShareKit"]
        ),
    ]
)
