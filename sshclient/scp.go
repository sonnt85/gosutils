// Full implement scp client mode for ssh
package sshclient

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	gofilepath "github.com/sonnt85/gofilepath"
	"golang.org/x/crypto/ssh"

	"strconv"
	"strings"

	"github.com/sonnt85/gosutils/bufcopy"
	log "github.com/sonnt85/gosutils/slogrus"
	"github.com/sonnt85/gosutils/sreflect"
	"github.com/sonnt85/gosutils/sutils"
	"github.com/sonnt85/gosystem"
	//	"strings"
)

type SecureCopier struct {
	IsRecursive bool
	IsQuiet     bool
	IsVerbose   bool
	//	inPipe      io.Reader
	//	outPipe     io.Writer
	//	errPipe     io.Writer
	ignErr  bool
	srcFile string
	dstFile string
}

// *ssh.Session
type ScpSession interface {
	WriteCloser() (io.WriteCloser, error)
	Reader() (io.Reader, error)
	Run(string) error
	Close() error
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
					log.WarnfS("scp processDir error [local ignore]: %v", err)
				} else {
					return err
				}
			}
		} else {
			err = scp.sendFile(procWriter, filepath.Join(srcFilePath, fi.Name()), fi)
			if err != nil {
				if scp.ignErr {
					log.WarnfS("scp sendFile error [local ignore]: %v", err)
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
		log.PrintfS("Sending end dir: %s", header)
	}
	_, err := procWriter.Write([]byte(header))
	return err
}

func (scp *SecureCopier) sendDir(procWriter io.Writer, srcPath string, srcFileInfo os.FileInfo) error {
	mode := uint32(srcFileInfo.Mode().Perm())
	header := fmt.Sprintf("D%04o 0 %s\n", mode, filepath.Base(srcPath))
	if scp.IsVerbose {
		log.InfofS("Sending Dir header : %s", header)
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
		log.InfofS("Sending File header: %s", header)
	}
	pb := sutils.NewProgressBar(srcPath, size)
	pb.Update(0)
	_, err = procWriter.Write([]byte(header))
	if err != nil {
		return err
	}
	//TODO buffering

	_, err = bufcopy.Copy(procWriter, fileReader)
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
		log.InfoS("Sent file plus null-byte.")
	}
	pb.Update(size)
	fmt.Println()

	if err != nil {
		log.ErrorS(err.Error())
	}
	return err
}

// to-scp [send scp -t ]
func scpToRemote(scp *SecureCopier, session ScpSession) error {

	srcFileInfo, err := os.Stat(scp.srcFile)
	if err != nil {
		log.ErrorS("Could not stat source file ", scp.srcFile)
		return err
	}

	if scp.IsVerbose {
		log.InfoS("Got session")
	}
	defer session.Close()
	ce := make(chan error, 1)
	if scp.dstFile == "" {
		scp.dstFile = filepath.Base(scp.srcFile)
		//scp.dstFile = "."
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		procWriter, err := session.WriteCloser()
		if err != nil {
			log.ErrorS(err.Error())
			ce <- err
			return
		}
		defer func() {
			err = procWriter.Close()
			if err != nil {
				log.ErrorS(err.Error())
				ce <- err
				return
			}
		}()

		if scp.IsRecursive {
			if srcFileInfo.IsDir() {
				err = scp.processDir(procWriter, scp.srcFile, srcFileInfo)
				if err != nil {
					if scp.ignErr {
						log.WarnfS("scp error [ignore]: %v", err)
					} else {
						log.ErrorS(err.Error())
						ce <- err
						return
					}
				}
			} else {
				err = scp.sendFile(procWriter, scp.srcFile, srcFileInfo)
				if err != nil {
					log.ErrorS(err.Error())
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
					log.ErrorS(err.Error())
					ce <- err
					return
				}
			}
		}
	}()
	// go func() {
	// 	select {
	// 	case err, ok := <-ce:
	// 		if err != nil { //ce is closed
	// 			log.ErrorS("Scp to server error:", err, ok)
	// 		} else {
	// 			session.Close()
	// 		}
	// 	}
	// }()

	remoteOpts := "-t"
	if scp.IsQuiet {
		remoteOpts += "-q"
	}
	if scp.IsRecursive {
		remoteOpts += "-r"
	}
	err = session.Run("scp " + remoteOpts + " " + scp.dstFile)
	if err != nil {
		log.ErrorS("Failed to run remote scp: ", err.Error())
	}
	wg.Wait()
	// time.Sleep(time.Second * 10)
	close(ce)
	return err
}

type ScpSessionFromsshSecsion struct {
	*ssh.Session
}

func (s *ScpSessionFromsshSecsion) Run(cmd string) error {
	return s.Session.Run(cmd)
}

func (s *ScpSessionFromsshSecsion) Close() error {
	return s.Session.Close()
}

func (s *ScpSessionFromsshSecsion) WriteCloser() (io.WriteCloser, error) {
	return s.Session.StdinPipe()
}

func (s *ScpSessionFromsshSecsion) Reader() (io.Reader, error) {
	return s.Session.StdoutPipe()
}

func NewScpSessionFromsshSecsion(session *ssh.Session) ScpSession {
	return &ScpSessionFromsshSecsion{session}
}

// scp FROM remote source[ send scp -f]
func scpFromRemote(scp *SecureCopier, session ScpSession) error {
	dstDir := scp.dstFile
	var useSpecifiedFilename bool
	var err error

	if strings.HasSuffix(scp.dstFile, string(os.PathSeparator)) {
		dstDir = scp.dstFile
		useSpecifiedFilename = false
	} else {
		dstDir = filepath.Dir(scp.dstFile)
		useSpecifiedFilename = true
	}

	//from-scp
	if scp.IsVerbose {
		log.InfoS("Got session")
	}
	//	defer session.Close()
	ce := make(chan error, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		cw, err := session.WriteCloser()
		if err != nil {
			log.ErrorS(err.Error())
			ce <- err
			return
		}
		defer func() {
			cw.Close()
		}()
		r, err := session.Reader()
		if err != nil {
			log.ErrorS("session stdout err: " + err.Error() + " continue anyway")
			ce <- err
			return
		}
		if scp.IsVerbose {
			log.InfoS("Sending null byte")
		}
		err = sendByte(cw, 0)
		if err != nil {
			log.ErrorS("Write error: " + err.Error())
			ce <- err
			return
		}
		//defer r.Close()
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
			log.WarnfS("[%s] Reading stdin of scp secssion [ max %d bytes ]: ....", desc, nb)

			n, err := r.Read(cmdArr)
			if err != nil {
				log.ErrorS("Error reading standard input:", err)
			} else {
				log.PrintfS("Dump data stdin of scp secssion [%d/%d]:\n%s", n, nb, hex.Dump(cmdArr))
			}
		}
		//	scploop:
		for more {
			cntloop = cntloop + 1

			cmdArr := make([]byte, 1)
			//			log.ErrorS("\nSCPloop times: ", cntloop)
			n, err := r.Read(cmdArr)

			if err != nil {
				//				log.ErrorfS("r.Read(cmdArr): %v", err)
				if err == io.EOF {
					//no problem.
					if scp.IsVerbose {
						log.InfoS("Received EOF from remote server")
					}
				} else {
					log.ErrorS("Error reading standard input:", err)
					ce <- err
				}
				return
			}
			if n < 1 {
				log.ErrorS("Error reading next byte from standard input")
				ce <- errors.New("error reading next byte from standard input")
				return
			}

		from0x1:
			cmd := cmdArr[0]
			if scp.IsVerbose {
				log.PrintfS("Sink cmd: %s (%v)\n", string(cmd), cmd)
			}
			switch cmd {
			case 0x0:
				//continue
				if scp.IsVerbose {
					log.InfoS("Received OK \n")
				}
			case 0xA: //newline
				//0xA command: end?

				if scp.IsVerbose {
					log.PrintS("Received All-done [0xA command]")
				}

				err = sendByte(cw, 0)
				if err != nil {
					log.ErrorS("Write error: " + err.Error())
					ce <- err
				}

				return
			case 0x1, 'D', 'C', 'E':
				//				if true && cntloop == 100 {
				//					cmdArrs := make([]byte, 128)
				//					n, _ := r.Read(cmdArrs)
				//					log.WarnfS("Debug data at loop %d [%d]:\n%s", cntloop, n, hex.Dump(cmdArrs))
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
								log.InfoS("Received EOF from remote server")
							}
						} else {
							log.ErrorS("Error reading standard input:", err)
							ce <- err
						}

						return
					}
					//first line

					cmdFull = scanner.Text()
				}
				//				log.InfofS("scanner.Bytes:\n%s", hex.Dump([]byte(cmdFull)))
				if scp.IsVerbose {
					log.InfofS("Sink Details [data only]: %v\n", cmdFull)
				}
				//remainder, split by spaces
				parts := strings.SplitN(cmdFull, " ", 3)
				//				re := regexp.New(`^([^ ]+) (.+) ([^ ]+)$`)
				//				re.Regexp() //compile
				//				parts := re.FindStringSubmatch(cmdFull)[1:]
				//				parts := re.SubexpNames()
				//				log.InfoS(parts)
				switch cmd {
				case 'E':
					//				if cntloop == 6 {
					captureStdIn("========================>", 0)
					//				}
					//E command: go back out of dir
					dstDir = filepath.Dir(dstDir)
					if scp.IsVerbose {
						//					log.InfoS("Entering directory: ", thisDstFile)
						log.InfoS("Received End-Dir, go back out of dir to: ", dstDir)
					}
					err = sendByte(cw, 0)
					if err != nil {
						log.ErrorfS("Write error: %s", err.Error())
						ce <- err
						return
					}
				case 0x1:
					if scp.ignErr {
						//						err = sendByte(cw, 0)
						//						if err != nil {
						//							log.ErrorS("Write error: " + err.Error())
						//							ce <- err
						//						}
						fmt.Println()
						log.ErrorfS("Received error message from server for 0x1[ignore]: %v\n", cmdFull[1:])
						scanner.Scan()
						err := scanner.Err()
						if err != nil {
							if err == io.EOF {
								//no problem.
								if scp.IsVerbose {
									log.InfoS("Received EOF from remote server")
								}
							} else {
								log.ErrorS("Error reading standard input:", err)
								ce <- err
							}

							return
						}

						jumfrom0x1 = true
						cmdArr[0] = scanner.Text()[0]
						goto from0x1
						//						continue
					} else {
						log.ErrorfS("Received error message: %v\n", cmdFull[1:])
						ce <- errors.New(cmdFull[1:])
						return
					}
				case 'D', 'C':
					mode, err := strconv.ParseInt(parts[0], 8, 32)

					if err != nil {
						log.ErrorS("Format error: " + err.Error())
						ce <- err
						return
					}
					sizeUint, err := strconv.ParseUint(parts[1], 10, 64)
					size := int64(sizeUint)
					if err != nil {
						log.ErrorS("Format error: " + err.Error())
						ce <- err
						return
					}
					rcvFilename := parts[2]
					if scp.IsVerbose {
						log.InfofS("Mode: %04o, size: %d, filename: %s\n", mode, size, rcvFilename)
					}
					var filename string
					//use the specified filename from the destination (only for top-level item)
					if useSpecifiedFilename && first {
						filename = filepath.Base(scp.dstFile)
					} else {
						filename = rcvFilename
					}
					err = sendByte(cw, 0)
					if err != nil {
						log.ErrorS("Send error: " + err.Error())
						ce <- err
						return
					}
					if cmd == 'C' {
						//C command - file
						thisDstFile := filepath.Join(dstDir, filename)
						tmpDstFile := sutils.TempFileCreateInNewTemDir(filename)
						defer os.RemoveAll(filepath.Dir(tmpDstFile))
						if scp.IsVerbose {
							log.InfoS("Creating destination file: ", thisDstFile)
						}
						tot := int64(0)
						pb := sutils.NewProgressBar(filename, size)
						pb.Update(0)

						fw, err := os.Create(tmpDstFile) //TODO: mode here
						//						fw, err := os.OpenFile(thisDstFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(mode))
						if err != nil {
							ce <- err
							log.ErrorS("File creation error: " + err.Error())
							return
						}

						defer fw.Close()

						//buffered by 4096 bytes
						bufferSize := int64(4096)
						lastPercent := int64(0)
						for tot < size {
							if bufferSize > size-tot {
								bufferSize = size - tot
							}
							b := make([]byte, bufferSize)
							n, err = r.Read(b)
							if err != nil {
								log.ErrorS("Read error: " + err.Error())
								ce <- err
								return
							}
							tot += int64(n)
							//write to file
							_, err = fw.Write(b[:n])
							if err != nil {
								log.ErrorS("Write error: " + err.Error())
								ce <- err
								return
							}
							percent := (100 * tot) / size
							if percent > lastPercent {
								pb.Update(tot)
							}
							lastPercent = percent
						}
						err = fw.Close()
						if err != nil {
							log.ErrorS(err.Error())
							ce <- err
							return
						}

						err = os.Rename(tmpDstFile, thisDstFile)
						gosystem.Chmod(thisDstFile, fs.FileMode(mode)) //Need test

						if err != nil {
							log.ErrorS(err.Error())
							ce <- err
							return
						}
						//						sutils.FileCopy(tmpDstFile, thisDstFile)
						//close file writer & check error

						//get next byte from channel reader
						nb := make([]byte, 1)
						_, err = r.Read(nb)
						if err != nil {
							log.ErrorS(err.Error())
							ce <- err
							return
						}
						//TODO check value received in nb
						//send null-byte back
						_, err = cw.Write([]byte{0})
						if err != nil {
							log.ErrorS("Send null-byte error: " + err.Error())
							ce <- err
							return
						}
						pb.Update(tot)
						fmt.Println() //new line
					} else if cmd == 'D' {
						//D command (directory)
						thisDstFile := filepath.Join(dstDir, filename)
						fileMode := os.FileMode(uint32(mode))
						err = os.MkdirAll(thisDstFile, fileMode)
						if err != nil {
							log.ErrorS("Mkdir error: " + err.Error())
							ce <- err
							return
						} else {
							if scp.IsVerbose {
								log.InfoS("Entering directory: ", thisDstFile)
							}
						}
						dstDir = thisDstFile
					}
				}
			default:
				log.WarnfS("Command '%v' NOT implementented\n", cmd)
				return
			}
			first = false
		}
		err = cw.Close()
		if err != nil {
			log.ErrorS("error closing process writer: ", err.Error())
			ce <- err
			return
		}
	}()

	// go func() {
	// 	select {
	// 	case err, ok := <-ce:
	// 		if err != nil { //ce is closed
	// 			log.ErrorS("Scp from remote error:", err, ok)
	// 		} else {
	// 			session.Close()
	// 		}
	// 	}
	// }()
	//qprf
	remoteOpts := "-f"
	if scp.IsQuiet {
		remoteOpts += "-q"
	}
	if scp.IsRecursive {
		remoteOpts += "-r"
	}
	//TODO should this path (/usr/bin/scp) be configurable?
	err = session.Run("scp " + remoteOpts + " " + scp.srcFile)
	if err != nil {
		fmt.Println()
		log.ErrorS("Failed to run remote scp: " + err.Error())
	} else {
		log.InfoS("Done scp")
	}
	wg.Wait()
	close(ce)
	return err

}

type ScpSessionFromReadWriteCloser struct {
	wc  io.WriteCloser
	r   io.Reader
	run func(string) error
}

func (s *ScpSessionFromReadWriteCloser) WriteCloser() (io.WriteCloser, error) {
	return s.wc, nil
}

func (s *ScpSessionFromReadWriteCloser) Reader() (io.Reader, error) {
	return s.r, nil
}

func (s *ScpSessionFromReadWriteCloser) Run(cmd string) error {
	if s.run == nil {
		return nil
	}
	return s.run(cmd)
}

func (s *ScpSessionFromReadWriteCloser) Close() error {
	return nil
	// return s.wc.Close()
}

func NewScpSessionFromReadWriteCloser(wc io.WriteCloser, r io.Reader, run func(string) error) ScpSession {
	return &ScpSessionFromReadWriteCloser{wc, r, run}
}

func SCP(inPipe io.WriteCloser, outPipe io.Reader, run func(string) error, srcFile string, dstFile string, commands ...string) (err error) {
	scp := &SecureCopier{
		srcFile: gofilepath.FromSlashSmart(srcFile, true),
		dstFile: gofilepath.FromSlashSmart(dstFile, true),
	}
	if sreflect.SlideHasElem(commands, "-r") || strings.HasSuffix(srcFile, string(os.PathSeparator)) {
		scp.IsRecursive = true
	} else {
		scp.IsRecursive = false
	}

	if sreflect.SlideHasElem(commands, "-q") {
		scp.IsQuiet = true
	} else {
		scp.IsQuiet = false
	}
	scp.IsVerbose = !scp.IsQuiet
	scp.ignErr = false
	session := NewScpSessionFromReadWriteCloser(inPipe, outPipe, run)
	if sreflect.SlideHasElem(commands, "-t") {
		// scp.dstFile = gofilepath.FromSlashSmart(commands[len(commands)-1], true)
		err = scpToRemote(scp, session)
		return
	}
	if sreflect.SlideHasElem(commands, "-f") {
		// scp.srcFile = gofilepath.FromSlashSmart(commands[len(commands)-1], true)
		err = scpFromRemote(scp, session)
		return
	}
	return nil
}
