class Sage < Formula
    desc "Encrypt files to a github user's SSH key"
    homepage "https://github.com/ryan-gerstenkorn-sp/bctx"
    url "https://github.com/ryan-gerstenkorn-sp/bctx.git"
    version "0.5.0"

    depends_on "go" => :build

    def install
      ENV["GOPATH"] = HOMEBREW_CACHE/"go_cache"
      mkdir bin
      system "go", "build", "-trimpath", "-o", bin, "ryan-gerstenkorn-sp/bctx"
      prefix.install_metafiles
    end

    test do
        system "false"
    end
end

