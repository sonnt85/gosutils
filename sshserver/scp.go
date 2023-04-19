// Full implement scp server mode for ssh

package sshserver

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"

	filepath "github.com/sonnt85/gofilepath"
	"github.com/sonnt85/gosutils/slogrus"
	"github.com/sonnt85/gosutils/sutils"
	"github.com/sonnt85/gosystem"
)

type SecureCopier struct {
	IsRecursive bool
	IsQuiet     bool
	IsVerbose   bool
	inPipe      io.WriteCloser
	outPipe     io.ReadCloser
	//	errPipe     io.Writer
	ignErr  bool
	srcFile string
	dstFile string
}

func sendByte(w io.Writer, val byte) error {
	_, err := w.Write([]byte{val})
	return err
}

//copy to server

func (scp *SecureCopier) Name() string {
	return "scp"
}
func (scp *SecureCopier) processDir(procWriter io.Writer, srcFilePath string, srcFileInfo os.FileInfo) error {
	err := scp.sendDir(procWriter, srcFilePath, srcFileInfo)
	if err != nil {
		return err
	}
	dir, err := os.Open(srcFilePath)
	if err != nil {
		return err
	}
	fis, err := dir.Readdir(0)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		if fi.IsDir() {
			err = scp.processDir(procWriter, filepath.Join(srcFilePath, fi.Name()), fi)
			if err != nil {
				if scp.ignErr {
					slogrus.WarnfS("scp processDir error [local ignore]: %v", err)
				} else {
					return err
				}
			}
		} else {
			err = scp.sendFile(procWriter, filepath.Join(srcFilePath, fi.Name()), fi)
			if err != nil {
				if scp.ignErr {
					slogrus.WarnfS("scp sendFile error [local ignore]: %v", err)
				} else {
					return err
				}
			}
		}
	}
	//TODO process errors
	err = scp.sendEndDir(procWriter)
	return err
}

func (scp *SecureCopier) sendEndDir(procWriter io.Writer) error {
	header := "E\n"
	if scp.IsVerbose {
		slogrus.PrintfS("Sending end dir: %s", header)
	}
	_, err := procWriter.Write([]byte(header))
	return err
}

func (scp *SecureCopier) sendDir(procWriter io.Writer, srcPath string, srcFileInfo os.FileInfo) error {
	mode := uint32(srcFileInfo.Mode().Perm())
	header := fmt.Sprintf("D%04o 0 %s\n", mode, filepath.Base(srcPath))
	if scp.IsVerbose {
		slogrus.InfoS("Sending Dir header : %s", header)
	}
	_, err := procWriter.Write([]byte(header))
	return err
}

func (scp *SecureCopier) sendFile(procWriter io.Writer, srcPath string, srcFileInfo os.FileInfo) error {
	//single file
	mode := uint32(srcFileInfo.Mode().Perm())
	fileReader, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer fileReader.Close()
	size := srcFileInfo.Size()
	header := fmt.Sprintf("C%04o %d %s\n", mode, size, filepath.Base(srcPath))
	if scp.IsVerbose {
		slogrus.PrintS("Sending File header: %s", header)
	}
	pb := sutils.NewProgressBar(srcPath, size)
	pb.Update(0)
	_, err = procWriter.Write([]byte(header))
	if err != nil {
		return err
	}
	//TODO buffering
	_, err = io.Copy(procWriter, fileReader)
	if err != nil {
		return err
	}
	// terminate with null byte
	err = sendByte(procWriter, 0)
	if err != nil {
		return err
	}

	err = fileReader.Close()
	if scp.IsVerbose {
		slogrus.PrintS("Sent file plus null-byte.")
	}
	pb.Update(size)
	slogrus.PrintS()

	if err != nil {
		slogrus.ErrorS(err.Error())
	}
	return err
}

// client send -f
func scpToClient(scp *SecureCopier) error {

	srcFileInfo, err := os.Stat(scp.srcFile)
	if err != nil {
		slogrus.ErrorS("Could not stat source file " + scp.srcFile)
		return err
	}
	if err != nil {
		return err
	} else if scp.IsVerbose {
		slogrus.InfoS("Got session")
	}
	//	defer session.Close()
	ce := make(chan error, 1)
	if scp.dstFile == "" {
		scp.dstFile = filepath.Base(scp.srcFile)
		//scp.dstFile = "."
	}
	var ferr error
	go func() {
		var ok bool
		ferr, ok = <-ce
		if ferr != nil { //ce is closed
			slogrus.ErrorS("Scp to client error:", ferr, ok)
		}
		// else {
		//				session.Close()
		// }
	}()
	func() {
		procWriter := scp.inPipe.(io.Writer)
		defer scp.inPipe.Close()
		if scp.IsRecursive {
			if srcFileInfo.IsDir() {
				err = scp.processDir(procWriter, scp.srcFile, srcFileInfo)
				if err != nil {
					if scp.ignErr {
						slogrus.WarnfS("scp error [ignore]: %v", err)
					} else {
						slogrus.ErrorS(err.Error())
						ce <- err
						return
					}
				}
			} else {
				err = scp.sendFile(procWriter, scp.srcFile, srcFileInfo)
				if err != nil {
					slogrus.ErrorS(err.Error())
					ce <- err
					return
				}
			}
		} else {
			if srcFileInfo.IsDir() {
				ce <- errors.New("error: Not a regular file")
				return
			} else {
				err = scp.sendFile(procWriter, scp.srcFile, srcFileInfo)
				if err != nil {
					slogrus.ErrorS(err.Error())
					ce <- err
					return
				}
			}
		}
		err = scp.inPipe.Close()
		if err != nil {
			slogrus.ErrorS(err.Error())
			ce <- err
			return
		}
	}()
	close(ce)
	return ferr
}

// Client send to server (scp -t)
func scpFromClient(scp *SecureCopier) error {
	slogrus.InfoS("Running scpFromClient")

	dstDir := scp.dstFile
	var useSpecifiedFilename bool
	if strings.HasSuffix(scp.dstFile, string(os.PathSeparator)) {
		dstDir = scp.dstFile
		useSpecifiedFilename = false
	} else {
		dstDir = filepath.Dir(scp.dstFile)
		useSpecifiedFilename = true
	}

	//from-scp
	//	defer session.Close()
	ce := make(chan error, 1)
	var ferr error
	//wait error
	go func() {
		var ok bool
		ferr, ok = <-ce
		if ferr != nil { //ce is closed
			slogrus.ErrorS("Scp from client error:", ferr, ok)
		}
		//  else {
		//				session.Close()
		// }
	}()
	func() {
		//		cw, err := session.(io.ReadCloser)
		w, ok := scp.inPipe.(io.Writer)
		if !ok {
			err := fmt.Errorf("not impliment interface writer")
			slogrus.ErrorS(err.Error())
			ce <- err
			return
		}
		defer scp.inPipe.Close()
		r, ok := scp.outPipe.(io.Reader)
		if !ok {
			err := fmt.Errorf("not impliment interface reader")
			slogrus.ErrorS("session stdout err: " + err.Error() + " continue anyway")
			ce <- err
			return
		}
		if scp.IsVerbose {
			slogrus.PrintS("Sending null byte")
		}

		if err := sendByte(w, 0); err != nil {
			slogrus.ErrorS("Write error: " + err.Error())
			ce <- err
			return
		}
		//		defer scp.outPipe.Close()
		//use a scanner for processing individual commands, but not files themselves
		scanner := bufio.NewScanner(r)
		scanner.Split(bufio.ScanLines)
		more := true
		first := true
		cntloop := 0
		jumfrom0x1 := false

		captureStdIn := func(desc string, nb int) {
			if nb == 0 {
				return
			}
			cmdArr := make([]byte, nb)
			slogrus.WarnfS("[%s] Reading stdin of scp secssion [ max %d bytes ]: ....", desc, nb)

			n, err := r.Read(cmdArr)
			if err != nil {
				slogrus.ErrorS("Error reading standard input:", err)
			} else {
				slogrus.PrintfS("Dump data stdin of scp secssion [%d/%d]:\n%s", n, nb, hex.Dump(cmdArr))
			}
		}
		//	scploop:
		for more {
			cntloop = cntloop + 1

			cmdArr := make([]byte, 1)
			//slogrus.ErrorS("\nSCPloop times: ", cntloop)
			n, err := r.Read(cmdArr)

			if err != nil {
				//slogrus.ErrorfS("r.Read(cmdArr): %v", err)
				if err == io.EOF {
					//no problem.
					if scp.IsVerbose {
						slogrus.PrintS("Received EOF from remote server")
					}
				} else {
					slogrus.ErrorS("Error reading standard input:", err)
					ce <- err
				}
				return
			}
			if n < 1 {
				slogrus.ErrorS("Error reading next byte from standard input")
				ce <- errors.New("error reading next byte from standard input")
				return
			}

		from0x1:
			cmd := cmdArr[0]
			if scp.IsVerbose {
				slogrus.PrintfS("Sink cmd: %s (%v)\n", string(cmd), cmd)
			}
			switch cmd {
			case 0x0:
				//continue
				if scp.IsVerbose {
					slogrus.PrintS("Received OK \n")
				}
			case 0xA: //newline
				//0xA command: end?

				if scp.IsVerbose {
					slogrus.PrintS("Received All-done [0xA command]\n")
				}

				err = sendByte(w, 0)
				if err != nil {
					slogrus.ErrorS("Write error: " + err.Error())
					ce <- err
				}

				return
			case 0x1, 'D', 'C', 'E':
				//				if true && cntloop == 100 {
				//					cmdArrs := make([]byte, 128)
				//					n, _ := r.Read(cmdArrs)
				//					slogrus.WarnfS("Debug data at loop %d [%d]:\n%s", cntloop, n, hex.Dump(cmdArrs))
				//				}
				cmdFull := ""
				if jumfrom0x1 {
					cmdFull = scanner.Text()[1:]
					jumfrom0x1 = false
				} else {
					scanner.Scan()
					err = scanner.Err()
					if err != nil {
						if err == io.EOF {
							//no problem.
							if scp.IsVerbose {
								slogrus.InfoS("Received EOF from remote server")
							}
						} else {
							slogrus.ErrorS("Error reading standard input:", err)
							ce <- err
						}

						return
					}
					//first line

					cmdFull = scanner.Text()
				}
				//				slogrus.Infof("scanner.Bytes:\n%s", hex.Dump([]byte(cmdFull)))
				if scp.IsVerbose {
					slogrus.InfofS("Sink Details [data only]: %v\n", cmdFull)
				}
				//remainder, split by spaces
				parts := strings.SplitN(cmdFull, " ", 3)
				//				re := regexp.New(`^([^ ]+) (.+) ([^ ]+)$`)
				//				re.Regexp() //compile
				//				parts := re.FindStringSubmatch(cmdFull)[1:]
				//				parts := re.SubexpNames()
				//				slogrus.PrintS(parts)
				switch cmd {
				case 'E':
					//				if cntloop == 6 {
					captureStdIn("========================>", 0)
					//				}
					//E command: go back out of dir
					dstDir = filepath.Dir(dstDir)
					if scp.IsVerbose {
						//					slogrus.Info("Entering directory: ", thisDstFile)
						slogrus.PrintS("Received End-Dir, go back out of dir to: ", dstDir)
					}
					err = sendByte(w, 0)
					if err != nil {
						slogrus.ErrorfS("Write error: %s", err.Error())
						ce <- err
						return
					}
				case 0x1:
					if scp.ignErr {
						//						err = sendByte(cw, 0)
						//						if err != nil {
						//							slogrus.ErrorS("Write error: " + err.Error())
						//							ce <- err
						//						}
						slogrus.PrintS()
						slogrus.ErrorfS("Received error message from server for 0x1[ignore]: %v\n", cmdFull[1:])
						scanner.Scan()
						err := scanner.Err()
						if err != nil {
							if err == io.EOF {
								//no problem.
								if scp.IsVerbose {
									slogrus.InfoS("Received EOF from remote server")
								}
							} else {
								slogrus.ErrorS("Error reading standard input:", err)
								ce <- err
							}

							return
						}

						jumfrom0x1 = true
						cmdArr[0] = scanner.Text()[0]
						goto from0x1
						//						continue
					} else {
						slogrus.ErrorfS("Received error message: %v\n", cmdFull[1:])
						ce <- errors.New(cmdFull[1:])
						return
					}
				case 'D', 'C':
					mode, err := strconv.ParseInt(parts[0], 8, 32)

					if err != nil {
						slogrus.ErrorS("Format error: " + err.Error())
						ce <- err
						return
					}
					sizeUint, err := strconv.ParseUint(parts[1], 10, 64)
					size := int64(sizeUint)
					if err != nil {
						slogrus.ErrorS("Format error: " + err.Error())
						ce <- err
						return
					}
					rcvFilename := parts[2]
					if scp.IsVerbose {
						slogrus.InfofS("Mode: %04o, size: %d, filename: %s\n", mode, size, rcvFilename)
					}
					var filename string
					//use the specified filename from the destination (only for top-level item)
					if useSpecifiedFilename && first {
						filename = filepath.Base(scp.dstFile)
					} else {
						filename = rcvFilename
					}
					err = sendByte(w, 0)
					if err != nil {
						slogrus.ErrorS("Send error: " + err.Error())
						ce <- err
						return
					}
					if cmd == 'C' {
						//C command - file
						thisDstFile := filepath.Join(dstDir, filename)
						if sutils.PathIsFile(thisDstFile) {
							if !gosystem.FileIWriteable(thisDstFile) {
								ce <- errors.New("Can not write to file " + thisDstFile)
								return
							}
						}
						tmpDstFile := sutils.TempFileCreateInNewTemDir(filename)
						defer os.RemoveAll(filepath.Dir(tmpDstFile))
						if scp.IsVerbose {
							slogrus.PrintS("Creating destination file: ", thisDstFile)
						}
						tot := int64(0)

						//fw, err := os.Create(thisDstFile) //TODO: mode here

						fw, err := os.Create(tmpDstFile) //TODO: mode here default 0666
						//						fw, err := os.OpenFile(thisDstFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(mode))
						if err != nil {
							ce <- err
							slogrus.ErrorS("File creation error: " + err.Error())
							return
						}

						defer fw.Close()

						//buffered by 4096 bytes
						bufferSize := int64(4096)
						for tot < size {
							if bufferSize > size-tot {
								bufferSize = size - tot
							}
							b := make([]byte, bufferSize)
							n, err = r.Read(b)
							if err != nil {
								slogrus.ErrorS("Read error: " + err.Error())
								ce <- err
								return
							}
							tot += int64(n)
							//write to file
							_, err = fw.Write(b[:n])
							if err != nil {
								slogrus.ErrorS("Write error: " + err.Error())
								ce <- err
								return
							}
						}
						err = fw.Close()
						if err != nil {
							slogrus.ErrorS(err.Error())
							ce <- err
							return
						}

						err = os.Rename(tmpDstFile, thisDstFile)
						gosystem.Chmod(thisDstFile, fs.FileMode(mode)) //Need test

						if err != nil {
							slogrus.ErrorS(err.Error())
							ce <- err
							return
						}
						//						sutils.FileCopy(tmpDstFile, thisDstFile)
						//close file writer & check error

						//get next byte from channel reader
						nb := make([]byte, 1)
						_, err = r.Read(nb)
						if err != nil {
							slogrus.ErrorS(err.Error())
							ce <- err
							return
						}
						//TODO check value received in nb
						//send null-byte back
						_, err = w.Write([]byte{0})
						if err != nil {
							slogrus.ErrorS("Send null-byte error: " + err.Error())
							ce <- err
							return
						}
						//						slogrus.Print() //new line
					} else if cmd == 'D' {
						//D command (directory)
						thisDstFile := filepath.Join(dstDir, filename)
						fileMode := os.FileMode(uint32(mode))
						err = os.MkdirAll(thisDstFile, fileMode)
						if err != nil {
							slogrus.ErrorS("Mkdir error: " + err.Error())
							ce <- err
							return
						} else {
							if scp.IsVerbose {
								slogrus.InfoS("Entering directory: ", thisDstFile)
							}
						}
						dstDir = thisDstFile
					}
				}
			default:
				slogrus.WarnfS("Command '%v' NOT implementented\n", cmd)
				return
			}
			first = false
		}

		if err := scp.inPipe.Close(); err != nil {
			slogrus.ErrorS("error closing process writer: ", err.Error())
			ce <- err
			return
		}
	}()

	close(ce)
	return ferr
}
