#include <iostream>
#include <vector>
#include <string>
#include <cstring>
#include <stdexcept>
#include <unistd.h>
#include <sys/socket.h>
#include <algorithm>

// Reader implements buffering for an io.Reader object.
class Reader {
public:
    static const size_t defaultBufSize = 4096;

    Reader(int fd, size_t size = defaultBufSize) 
        : fd_(fd), buf_(size), r_(0), w_(0) {
        if (size <= 0) {
            buf_.resize(defaultBufSize);
        }
    }

    // Read reads data into p.
    // It returns the number of bytes read into p.
    // The bytes are taken from at most one Read on the underlying Reader,
    // hence n may be less than len(p).
    ssize_t Read(char* p, size_t n) {
        if (n == 0) {
            return 0;
        }

        if (r_ == w_) {
            if (n >= buf_.size()) {
                // Large read, empty buffer.
                // Read directly into p to avoid copy.
                ssize_t received = recv(fd_, p, n, 0);
                if (received < 0) {
                     return -1; // Error
                }
                return received;
            }
            // One read.
            // if (fd != 0) { ... } // Not checking standard input special case
            fill();
            if (r_ == w_) {
                return 0; // EOF
            }
        }

        size_t n_copy = std::min(n, (size_t)(w_ - r_));
        std::memcpy(p, &buf_[r_], n_copy);
        r_ += n_copy;
        return n_copy;
    }

    // ReadByte reads and returns a single byte.
    // Returns -1 if no byte is available (EOF or error). Note: Int return type to signal EOF/error vs byte value.
    // In Go this returns (byte, error). Here we simulate with int.
    int ReadByte() {
        while (r_ == w_) {
            fill();
            if (r_ == w_) {
                return -1;
            }
        }
        unsigned char c = buf_[r_];
        r_++;
        return c;
    }

    // ReadString reads until the first occurrence of delim in the input,
    // returning a string containing the data up to and including the delimiter.
    // If ReadString encounters an error before finding a delimiter,
    // it returns the data read before the error and the error itself (conventionally).
    // Here we throw or return partial on EOF.
    std::string ReadString(char delim) {
        std::string result;
        while (true) {
            bool found = false;
            size_t i;
            for (i = r_; i < w_; ++i) {
                if (buf_[i] == delim) {
                    found = true;
                    break;
                }
            }

            if (found) {
                result.append(&buf_[r_], i - r_ + 1);
                r_ = i + 1;
                return result;
            }

            // Not found in current buffer
            if (r_ < w_) {
                result.append(&buf_[r_], w_ - r_);
                r_ = w_;
            }

            fill();
            if (r_ == w_) {
                // EOF
                break;
            }
        }
        return result;
    }

    // Buffered returns the number of bytes that can be read from the current buffer.
    int Buffered() const {
        return w_ - r_;
    }

private:
	/*
	 * buf_:  [ consumed | unread data | free space ]
	 * 			 0..r_-1   r_..w_-1      w_..end
	 */
    int fd_;
    std::vector<char> buf_;
    size_t r_, w_; // buf read and write positions

    void fill() {
        // Slide existing data to beginning. (Optional optimization: only if empty or mostly empty)
        if (r_ > 0) {
            if (r_ < w_) {
                std::memmove(&buf_[0], &buf_[r_], w_ - r_);
            }
            w_ -= r_;
            r_ = 0;
        }

        if (w_ >= buf_.size()) {
             // Should not happen if logic is correct and size > 0
             return; 
        }

        // Read new data: try to fill the rest of the buffer.
        ssize_t n = recv(fd_, &buf_[w_], buf_.size() - w_, 0);
        if (n > 0) {
            w_ += n;
        } else if (n == 0) {
            // EOF
        } else {
            // Error
            // For simplicity, treat as EOF or handle appropriately
        }
    }
};
