# dedup

`dedup` is a command-line tool written in Go that scans a directory for duplicate files and replaces the duplicates with symlinks to the original file. It calculates the SHA-256 hash of each file to identify duplicates and uses a worker pool to process files concurrently.

The binary shipped in this repo is a stripped, statically linked binary that was tested on a 2TB hardrive containing tens of thousands of software projects, many of which had the same or similar dependencies. It processed the entire file system in about 8 hrs. Resulting in over 25% increase to free space. YMMV.

## WARNING

This is intended to be used only on userspace files. Do not run this on / or /boot
Furthermore, it is ill advised to run this on /home or /home/user (many files in these locations are empty files that serve solely as timestamp locking mechanisms, but will have the same hash because they are empty, meaning that chaos will ensue).

## Features

- Scans a directory for duplicate files
- Calculates the SHA-256 hash for each file
- Replaces duplicate files with symlinks to the original file
- Uses a worker pool to process files concurrently
- Stores file hashes and paths in an SQLite database for efficient lookup

## Usage

Usage: dedup <path_to_directory>


For example, to scan the directory `/home/user/documents` for duplicates, run:

dedup /home/user/documents

## Building

To build the `dedup` program, run the following command in the root directory of the project:

go build -o dedup main.go


This will create an executable called `dedup` in the current directory.

## License

`dedup` is released under the GNU General Public License version 3 (GPLv3). A copy of the license can be found in the `LICENSE` file included in the project.

## Contributing

Contributions are welcome! If you would like to contribute to the project or report a bug, please open an issue or submit a pull request on the GitHub repository.

## WARNING

This is intended to be used only on userspace files. Do not run this on / or /boot
Furthermore, it is ill advised to run this on /home or /home/user (many files in these locations are empty files that serve solely as timestamp locking mechanisms, but will have the same hash because they are empty, meaning that chaos will ensue).

## Disclaimer

`dedup` is provided "as is" without warranty of any kind. Please use this tool at your own risk, and ensure that you have backups of your data before using it.

While we have tested this and found that it operates well on our test systems. We are not responsible if this program causes damage to your system including but not limited to damaging or erasing files, rendering your system non-responsive, bricking your computer or summoning Chuthulu and bringing about the end of the world.