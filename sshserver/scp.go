// Package scp provides a simple interface to copying files over a
// go.crypto/ssh session.
package sshserver

import (
	"fmt"
	"io"
	"os"

	//	"path"
	"bufio"
	"errors"

	//	"time"
	//	"github.com/laher/sshutils-go/sshconn"
	//	"github.com/sonnt85/gosutils/regexp"

	//	shellquote "github.com/sonnt85/gosutils/shellwords"
	"encoding/hex"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/sonnt85/gosutils/sutils"
	//	"strings"
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
					log.Warnf("scp processDir error [local ignore]: %v", err)
				} else {
					return err
				}
			}
		} else {
			err = scp.sendFile(procWriter, filepath.Join(srcFilePath, fi.Name()), fi)
			if err != nil {
				if scp.ignErr {
					log.Warnf("scp sendFile error [local ignore]: %v", err)
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
	header := fmt.Sprintf("E\n")
	if scp.IsVerbose {
		log.Printf("Sending end dir: %s", header)
	}
	_, err := procWriter.Write([]byte(header))
	return err
}

func (scp *SecureCopier) sendDir(procWriter io.Writer, srcPath string, srcFileInfo os.FileInfo) error {
	mode := uint32(srcFileInfo.Mode().Perm())
	header := fmt.Sprintf("D%04o 0 %s\n", mode, filepath.Base(srcPath))
	if scp.IsVerbose {
		log.Infoln("Sending Dir header : %s", header)
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
		log.Println("Sending File header: %s", header)
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
		log.Println("Sent file plus null-byte.")
	}
	pb.Update(size)
	fmt.Println()

	if err != nil {
		log.Errorln(err.Error())
	}
	return err
}

//client send -f
func scpToClient(scp *SecureCopier) error {

	srcFileInfo, err := os.Stat(scp.srcFile)
	if err != nil {
		log.Errorln("Could not stat source file " + scp.srcFile)
		return err
	}
	if err != nil {
		return err
	} else if scp.IsVerbose {
		log.Infoln("Got session")
	}
	//	defer session.Close()
	ce := make(chan error)
	if scp.dstFile == "" {
		scp.dstFile = filepath.Base(scp.srcFile)
		//scp.dstFile = "."
	}
	go func() {
		select {
		case err, ok := <-ce:
			if err != nil { //ce is closed
				log.Errorln("Scp to client error:", err, ok)
			} else {
				//				session.Close()
			}
		}
	}()
	func() {
		procWriter := scp.inPipe.(io.Writer)
		defer scp.inPipe.Close()
		if scp.IsRecursive {
			if srcFileInfo.IsDir() {
				err = scp.processDir(procWriter, scp.srcFile, srcFileInfo)
				if err != nil {
					if scp.ignErr {
						log.Warnf("scp error [ignore]: %v", err)
					} else {
						log.Errorln(err.Error())
						ce <- err
						return
					}
				}
			} else {
				err = scp.sendFile(procWriter, scp.srcFile, srcFileInfo)
				if err != nil {
					log.Errorln(err.Error())
					ce <- err
					return
				}
			}
		} else {
			if srcFileInfo.IsDir() {
				ce <- errors.New("Error: Not a regular file")
				return
			} else {
				err = scp.sendFile(procWriter, scp.srcFile, srcFileInfo)
				if err != nil {
					log.Errorln(err.Error())
					ce <- err
					return
				}
			}
		}
		err = scp.inPipe.Close()
		if err != nil {
			log.Errorln(err.Error())
			ce <- err
			return
		}
	}()
	close(ce)
	return err
}

//Client send to server (scp -t)
func scpFromClient(scp *SecureCopier) error {
	log.Info("Running scpFromClient")

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
	go func() {
		select {
		case err, ok := <-ce:
			ferr = err
			if err != nil { //ce is closed
				log.Errorln("Scp from client error:", err, ok)
			} else {
				//				session.Close()
			}
		}
	}()
	func() {
		//		cw, err := session.(io.ReadCloser)
		w, ok := scp.inPipe.(io.Writer)
		if !ok {
			err := fmt.Errorf("Not impliment interface writer")
			log.Errorln(err.Error())
			ce <- err
			return
		}
		defer scp.inPipe.Close()
		r, ok := scp.outPipe.(io.Reader)
		if !ok {
			err := fmt.Errorf("Not impliment interface reader")
			log.Errorln("session stdout err: " + err.Error() + " continue anyway")
			ce <- err
			return
		}
		if scp.IsVerbose {
			log.Println("Sending null byte")
		}

		if err := sendByte(w, 0); err != nil {
			log.Errorln("Write error: " + err.Error())
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
			log.Warnf("[%s] Reading stdin of scp secssion [ max %d bytes ]: ....", desc, nb)

			n, err := r.Read(cmdArr)
			if err != nil {
				log.Errorln("Error reading standard input:", err)
			} else {
				log.Printf("Dump data stdin of scp secssion [%d/%d]:\n%s", n, nb, hex.Dump(cmdArr))
			}
			return
		}
		//	scploop:
		for more {
			cntloop = cntloop + 1

			cmdArr := make([]byte, 1)
			//			log.Errorln("\nSCPloop times: ", cntloop)
			n, err := r.Read(cmdArr)

			if err != nil {
				//				log.Errorf("r.Read(cmdArr): %v", err)
				if err == io.EOF {
					//no problem.
					if scp.IsVerbose {
						log.Println("Received EOF from remote server")
					}
				} else {
					log.Errorln("Error reading standard input:", err)
					ce <- err
				}
				return
			}
			if n < 1 {
				log.Errorln("Error reading next byte from standard input")
				ce <- errors.New("Error reading next byte from standard input")
				return
			}

		from0x1:
			cmd := cmdArr[0]
			if scp.IsVerbose {
				log.Printf("Sink cmd: %s (%v)\n", string(cmd), cmd)
			}
			switch cmd {
			case 0x0:
				//continue
				if scp.IsVerbose {
					log.Println("Received OK \n")
				}
			case 0xA: //newline
				//0xA command: end?

				if scp.IsVerbose {
					log.Print("Received All-done [0xA command]\n")
				}

				err = sendByte(w, 0)
				if err != nil {
					log.Errorln("Write error: " + err.Error())
					ce <- err
				}

				return
			case 0x1, 'D', 'C', 'E':
				//				if true && cntloop == 100 {
				//					cmdArrs := make([]byte, 128)
				//					n, _ := r.Read(cmdArrs)
				//					log.Warnf("Debug data at loop %d [%d]:\n%s", cntloop, n, hex.Dump(cmdArrs))
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
								log.Infoln("Received EOF from remote server")
							}
						} else {
							log.Errorln("Error reading standard input:", err)
							ce <- err
						}

						return
					}
					//first line

					cmdFull = scanner.Text()
				}
				//				log.Infof("scanner.Bytes:\n%s", hex.Dump([]byte(cmdFull)))
				if scp.IsVerbose {
					log.Infof("Sink Details [data only]: %v\n", cmdFull)
				}
				//remainder, split by spaces
				parts := strings.SplitN(cmdFull, " ", 3)
				//				re := regexp.New(`^([^ ]+) (.+) ([^ ]+)$`)
				//				re.Regexp() //compile
				//				parts := re.FindStringSubmatch(cmdFull)[1:]
				//				parts := re.SubexpNames()
				//				log.Println(parts)
				switch cmd {
				case 'E':
					//				if cntloop == 6 {
					captureStdIn("========================>", 0)
					//				}
					//E command: go back out of dir
					dstDir = filepath.Dir(dstDir)
					if scp.IsVerbose {
						//					log.Infoln("Entering directory: ", thisDstFile)
						log.Println("Received End-Dir, go back out of dir to: ", dstDir)
					}
					err = sendByte(w, 0)
					if err != nil {
						log.Errorf("Write error: %s", err.Error())
						ce <- err
						return
					}
				case 0x1:
					if scp.ignErr {
						//						err = sendByte(cw, 0)
						//						if err != nil {
						//							log.Errorln("Write error: " + err.Error())
						//							ce <- err
						//						}
						fmt.Println()
						log.Errorf("Received error message from server for 0x1[ignore]: %v\n", cmdFull[1:])
						scanner.Scan()
						err := scanner.Err()
						if err != nil {
							if err == io.EOF {
								//no problem.
								if scp.IsVerbose {
									log.Infoln("Received EOF from remote server")
								}
							} else {
								log.Errorln("Error reading standard input:", err)
								ce <- err
							}

							return
						}

						jumfrom0x1 = true
						cmdArr[0] = scanner.Text()[0]
						goto from0x1
						//						continue
					} else {
						log.Errorf("Received error message: %v\n", cmdFull[1:])
						ce <- errors.New(cmdFull[1:])
						return
					}
				case 'D', 'C':
					mode, err := strconv.ParseInt(parts[0], 8, 32)

					if err != nil {
						log.Errorln("Format error: " + err.Error())
						ce <- err
						return
					}
					sizeUint, err := strconv.ParseUint(parts[1], 10, 64)
					size := int64(sizeUint)
					if err != nil {
						log.Errorln("Format error: " + err.Error())
						ce <- err
						return
					}
					rcvFilename := parts[2]
					if scp.IsVerbose {
						log.Infof("Mode: %04o, size: %d, filename: %s\n", mode, size, rcvFilename)
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
						log.Errorln("Send error: " + err.Error())
						ce <- err
						return
					}
					if cmd == 'C' {
						//C command - file
						thisDstFile := filepath.Join(dstDir, filename)
						tmpDstFile := sutils.TempFileCreateInNewTemDir(filename)
						defer os.RemoveAll(filepath.Dir(tmpDstFile))
						if scp.IsVerbose {
							log.Println("Creating destination file: ", thisDstFile)
						}
						tot := int64(0)

						//fw, err := os.Create(thisDstFile) //TODO: mode here

						fw, err := os.Create(tmpDstFile) //TODO: mode here default 0666
						//						fw, err := os.OpenFile(thisDstFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(mode))
						if err != nil {
							ce <- err
							log.Errorln("File creation error: " + err.Error())
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
								log.Errorln("Read error: " + err.Error())
								ce <- err
								return
							}
							tot += int64(n)
							//write to file
							_, err = fw.Write(b[:n])
							if err != nil {
								log.Errorln("Write error: " + err.Error())
								ce <- err
								return
							}
						}
						err = fw.Close()
						if err != nil {
							log.Errorln(err.Error())
							ce <- err
							return
						}

						err = os.Rename(tmpDstFile, thisDstFile)
						if err != nil {
							log.Errorln(err.Error())
							ce <- err
							return
						}
						//						sutils.FileCopy(tmpDstFile, thisDstFile)
						//close file writer & check error

						//get next byte from channel reader
						nb := make([]byte, 1)
						_, err = r.Read(nb)
						if err != nil {
							log.Errorln(err.Error())
							ce <- err
							return
						}
						//TODO check value received in nb
						//send null-byte back
						_, err = w.Write([]byte{0})
						if err != nil {
							log.Errorln("Send null-byte error: " + err.Error())
							ce <- err
							return
						}
						//						fmt.Println() //new line
					} else if cmd == 'D' {
						//D command (directory)
						thisDstFile := filepath.Join(dstDir, filename)
						fileMode := os.FileMode(uint32(mode))
						err = os.MkdirAll(thisDstFile, fileMode)
						if err != nil {
							log.Errorln("Mkdir error: " + err.Error())
							ce <- err
							return
						} else {
							if scp.IsVerbose {
								log.Infoln("Entering directory: ", thisDstFile)
							}
						}
						dstDir = thisDstFile
					}
				}
			default:
				log.Warnf("Command '%v' NOT implementented\n", cmd)
				return
			}
			first = false
		}

		if err := scp.inPipe.Close(); err != nil {
			log.Errorln("error closing process writer: ", err.Error())
			ce <- err
			return
		}
	}()

	close(ce)
	return ferr
}
