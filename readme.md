<img width="792" height="702" alt="image" src="https://github.com/user-attachments/assets/56b1ad61-9a1a-4506-b9cb-beb4a6d9fa17" />

# Install 32-bit development libraries
sudo dnf install glibc-devel.i686 libstdc++-devel.i686 libgcc.i686

# Install 32-bit X11 development libraries (for GUI apps)
sudo dnf install libX11-devel.i686 libXcursor-devel.i686 \
  libXi-devel.i686 libXinerama-devel.i686 libXrandr-devel.i686 \
  mesa-libGL-devel.i686

# Verify installation
ls -la /usr/include/gnu/stubs-32.h  # Should exist now



# Install all 32-bit X11 development libraries
sudo dnf install \
  libXxf86vm-devel.i686 \
  libX11-devel.i686 \
  libXcursor-devel.i686 \
  libXi-devel.i686 \
  libXinerama-devel.i686 \
  libXrandr-devel.i686 \
  mesa-libGL-devel.i686 \
  mesa-libGLU-devel.i686


# Build for current platform only
make

# Build everything
make all

# Build only for Linux
make linux

# Build only for Windows (requires mingw)
make windows

# Install locally
make install

# Create distribution package
make package
