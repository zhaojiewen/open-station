# Open Station Homebrew Formula
# Install via: brew tap xuhaiqing/open-station && brew install open-station

class OpenStation < Formula
  desc "Enterprise AI Gateway supporting OpenAI, Claude, Gemini, DeepSeek, GLM with MCP Protocol"
  homepage "https://github.com/xuhaiqing/open-station"
  version "1.0.0"  # Will be updated automatically
  license "MIT"
  head "https://github.com/xuhaiqing/open-station.git", branch: "main"

  # Platform-specific downloads
  on_macos do
    on_intel do
      url "https://github.com/xuhaiqing/open-station/releases/download/v#{version}/open-station-#{version}-darwin-amd64.tar.gz"
      sha256 "" # Will be filled after release
    end
    on_arm do
      url "https://github.com/xuhaiqing/open-station/releases/download/v#{version}/open-station-#{version}-darwin-arm64.tar.gz"
      sha256 "" # Will be filled after release
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/xuhaiqing/open-station/releases/download/v#{version}/open-station-#{version}-linux-amd64.tar.gz"
      sha256 "" # Will be filled after release
    end
    on_arm do
      if Hardware::CPU.arm_version == 7
        url "https://github.com/xuhaiqing/open-station/releases/download/v#{version}/open-station-#{version}-linux-armv7.tar.gz"
        sha256 "" # Will be filled after release
      else
        url "https://github.com/xuhaiqing/open-station/releases/download/v#{version}/open-station-#{version}-linux-arm64.tar.gz"
        sha256 "" # Will be filled after release
      end
    end
  end

  # Dependencies
  depends_on "go" => :build if build.head?

  def install
    if build.head?
      # Build from source for head builds
      system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/server"
    else
      # Install pre-built binary
      bin.install "bin/open-station"
    end

    # Install configuration
    (etc/"open-station").install Dir["configs/*"] if Dir.exist?("configs")

    # Create log directory
    (var/"log/open-station").mkpath
  end

  def post_install
    # Create default config if not exists
    config_file = etc/"open-station/config.yaml"
    unless config_file.exist?
      config_file.write <<~EOF
        server:
          port: 8080
          mode: release

        database:
          host: localhost
          port: 5432
          user: postgres
          password: postgres
          dbname: ai_gateway
          sslmode: disable

        redis:
          host: localhost
          port: 6379
          password: ""
          db: 0
          pool_size: 100

        logging:
          level: info
          format: json
          output: stdout

        rate_limit:
          default_user_rps: 20
          default_tenant_rps: 200
      EOF
    end
  end

  # Service definition for brew services
  service do
    run [bin/"open-station", "-config", etc/"open-station/config.yaml"]
    keep_alive true
    log_path var/"log/open-station.log"
    error_log_path var/"log/open-station.error.log"
    working_dir var
  end

  test do
    # Version test
    output = shell_output("#{bin}/open-station --version 2>&1")
    assert_match "Open Station", output

    # Config validation test
    system bin/"open-station", "-config", etc/"open-station/config.yaml", "-validate"
  end
end