package model

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/sjzar/chatlog/pkg/util"
)

type MediaMessage struct {
	Type      int64
	SubType   int
	MediaMD5  string
	MediaPath string
	Title     string
	Desc      string
	Content   string
	URL       string

	RecordInfo *RecordInfo

	ReferDisplayName string
	ReferUserName    string
	ReferCreateTime  time.Time
	ReferMessage     *MediaMessage

	Host string

	Message XMLMessage
}

func NewMediaMessage(_type int64, data string) (*MediaMessage, error) {

	__type, subType := util.SplitInt64ToTwoInt32(_type)

	m := &MediaMessage{
		Type:    __type,
		SubType: int(subType),
	}

	if _type == 1 {
		m.Content = data
		return m, nil
	}

	var msg XMLMessage
	err := xml.Unmarshal([]byte(data), &msg)
	if err != nil {
		return nil, err
	}

	m.Message = msg
	if err := m.parse(); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *MediaMessage) parse() error {

	switch m.Type {
	case 3:
		m.MediaMD5 = m.Message.Image.MD5
	case 43:
		m.MediaMD5 = m.Message.Video.RawMd5
	case 49:
		m.SubType = m.Message.App.Type
		switch m.SubType {
		case 5:
			m.Title = m.Message.App.Title
			m.URL = m.Message.App.URL
		case 6:
			m.Title = m.Message.App.Title
			m.MediaMD5 = m.Message.App.MD5
		case 19:
			m.Title = m.Message.App.Title
			m.Desc = m.Message.App.Des
			if m.Message.App.RecordItem == nil {
				break
			}
			recordInfo := &RecordInfo{}
			err := xml.Unmarshal([]byte(m.Message.App.RecordItem.CDATA), recordInfo)
			if err != nil {
				return err
			}
			m.RecordInfo = recordInfo
		case 57:
			m.Content = m.Message.App.Title
			if m.Message.App.ReferMsg == nil {
				break
			}
			subMsg, err := NewMediaMessage(m.Message.App.ReferMsg.Type, m.Message.App.ReferMsg.Content)
			if err != nil {
				break
			}
			m.ReferDisplayName = m.Message.App.ReferMsg.DisplayName
			m.ReferUserName = m.Message.App.ReferMsg.ChatUsr
			m.ReferCreateTime = time.Unix(m.Message.App.ReferMsg.CreateTime, 0)
			m.ReferMessage = subMsg
		}
	}

	return nil
}

func (m *MediaMessage) SetHost(host string) {
	m.Host = host
}

func (m *MediaMessage) String() string {
	switch m.Type {
	case 1:
		return m.Content
	case 3:
		return fmt.Sprintf("![图片](http://%s/image/%s)", m.Host, m.MediaMD5)
	case 34:
		return "[语音]"
	case 43:
		if m.MediaPath != "" {
			return fmt.Sprintf("![视频](http://%s/data/%s)", m.Host, m.MediaPath)
		}
		return fmt.Sprintf("![视频](http://%s/video/%s)", m.Host, m.MediaMD5)
	case 47:
		return "[动画表情]"
	case 49:
		switch m.SubType {
		case 5:
			return fmt.Sprintf("[链接|%s](%s)", m.Title, m.URL)
		case 6:
			return fmt.Sprintf("[文件|%s](http://%s/file/%s)", m.Title, m.Host, m.MediaMD5)
		case 8:
			return "[GIF表情]"
		case 19:
			if m.RecordInfo == nil {
				return "[合并转发]"
			}
			buf := strings.Builder{}
			for _, item := range m.RecordInfo.DataList.DataItems {
				buf.WriteString(item.SourceName + ": ")
				switch item.DataType {
				case "jpg":
					buf.WriteString(fmt.Sprintf("![图片](http://%s/image/%s)", m.Host, item.FullMD5))
				default:
					buf.WriteString(item.DataDesc)
				}
				buf.WriteString("\n")
			}
			return m.Content
		case 33, 36:
			return "[小程序]"
		case 57:
			if m.ReferMessage == nil {
				if m.Content == "" {
					return "[引用]"
				}
				return "> [引用]\n" + m.Content
			}
			buf := strings.Builder{}
			buf.WriteString("> ")
			if m.ReferDisplayName != "" {
				buf.WriteString(m.ReferDisplayName)
				buf.WriteString("(")
				buf.WriteString(m.ReferUserName)
				buf.WriteString(")")
			} else {
				buf.WriteString(m.ReferUserName)
			}
			buf.WriteString(" ")
			buf.WriteString(m.ReferCreateTime.Format("2006-01-02 15:04:05"))
			buf.WriteString("\n")
			buf.WriteString("> ")
			m.ReferMessage.SetHost(m.Host)
			buf.WriteString(strings.ReplaceAll(m.ReferMessage.String(), "\n", "\n> "))
			buf.WriteString("\n")
			buf.WriteString(m.Content)
			m.Content = buf.String()
			return m.Content
		case 63:
			return "[视频号]"
		case 87:
			return "[群公告]"
		case 2000:
			return "[转账]"
		case 2003:
			return "[红包封面]"
		default:
			return "[分享]"
		}
	case 50:
		return "[语音通话]"
	case 10000:
		return "[系统消息]"
	default:
		content := m.Content
		if len(content) > 120 {
			content = content[:120] + "<...>"
		}
		return fmt.Sprintf("Type: %d Content: %s", m.Type, content)
	}
}

type XMLMessage struct {
	XMLName xml.Name `xml:"msg"`
	Image   Image    `xml:"img,omitempty"`
	Video   Video    `xml:"videomsg,omitempty"`
	App     App      `xml:"appmsg,omitempty"`
}

type XMLImageMessage struct {
	XMLName xml.Name `xml:"msg"`
	Img     Image    `xml:"img"`
}

type Image struct {
	MD5 string `xml:"md5,attr"`
	// HdLength            string `xml:"hdlength,attr"`
	// Length              string `xml:"length,attr"`
	// AesKey              string `xml:"aeskey,attr"`
	// EncryVer            string `xml:"encryver,attr"`
	// OriginSourceMd5     string `xml:"originsourcemd5,attr"`
	// FileKey             string `xml:"filekey,attr"`
	// UploadContinueCount string `xml:"uploadcontinuecount,attr"`
	// ImgSourceUrl        string `xml:"imgsourceurl,attr"`
	// HevcMidSize         string `xml:"hevc_mid_size,attr"`
	// CdnBigImgUrl        string `xml:"cdnbigimgurl,attr"`
	// CdnMidImgUrl        string `xml:"cdnmidimgurl,attr"`
	// CdnThumbUrl         string `xml:"cdnthumburl,attr"`
	// CdnThumbLength      string `xml:"cdnthumblength,attr"`
	// CdnThumbWidth       string `xml:"cdnthumbwidth,attr"`
	// CdnThumbHeight      string `xml:"cdnthumbheight,attr"`
	// CdnThumbAesKey      string `xml:"cdnthumbaeskey,attr"`
}

type XMLVideoMessage struct {
	XMLName  xml.Name `xml:"msg"`
	VideoMsg Video    `xml:"videomsg"`
}

type Video struct {
	RawMd5 string `xml:"rawmd5,attr"`
	// Length            string `xml:"length,attr"`
	// PlayLength        string `xml:"playlength,attr"`
	// Offset            string `xml:"offset,attr"`
	// FromUserName      string `xml:"fromusername,attr"`
	// Status            string `xml:"status,attr"`
	// Compress          string `xml:"compress,attr"`
	// CameraType        string `xml:"cameratype,attr"`
	// Source            string `xml:"source,attr"`
	// AesKey            string `xml:"aeskey,attr"`
	// CdnVideoUrl       string `xml:"cdnvideourl,attr"`
	// CdnThumbUrl       string `xml:"cdnthumburl,attr"`
	// CdnThumbLength    string `xml:"cdnthumblength,attr"`
	// CdnThumbWidth     string `xml:"cdnthumbwidth,attr"`
	// CdnThumbHeight    string `xml:"cdnthumbheight,attr"`
	// CdnThumbAesKey    string `xml:"cdnthumbaeskey,attr"`
	// EncryVer          string `xml:"encryver,attr"`
	// RawLength         string `xml:"rawlength,attr"`
	// CdnRawVideoUrl    string `xml:"cdnrawvideourl,attr"`
	// CdnRawVideoAesKey string `xml:"cdnrawvideoaeskey,attr"`
}

type App struct {
	Type       int         `xml:"type"`
	Title      string      `xml:"title"`
	Des        string      `xml:"des"`
	URL        string      `xml:"url"`                  // type 5 分享
	AppAttach  AppAttach   `xml:"appattach"`            // type 6 文件
	MD5        string      `xml:"md5"`                  // type 6 文件
	RecordItem *RecordItem `xml:"recorditem,omitempty"` // type 19 合并转发
	ReferMsg   *ReferMsg   `xml:"refermsg,omitempty"`   // type 57 引用
}

// ReferMsg 表示引用消息
type ReferMsg struct {
	Type        int64  `xml:"type"`
	SvrID       string `xml:"svrid"`
	FromUsr     string `xml:"fromusr"`
	ChatUsr     string `xml:"chatusr"`
	DisplayName string `xml:"displayname"`
	MsgSource   string `xml:"msgsource"`
	Content     string `xml:"content"`
	StrID       string `xml:"strid"`
	CreateTime  int64  `xml:"createtime"`
}

// AppAttach 表示应用附件
type AppAttach struct {
	TotalLen       string `xml:"totallen"`
	AttachID       string `xml:"attachid"`
	CDNAttachURL   string `xml:"cdnattachurl"`
	EmoticonMD5    string `xml:"emoticonmd5"`
	AESKey         string `xml:"aeskey"`
	FileExt        string `xml:"fileext"`
	IsLargeFileMsg string `xml:"islargefilemsg"`
}

type RecordItem struct {
	CDATA string `xml:",cdata"`

	// 解析后的记录信息
	RecordInfo *RecordInfo
}

// RecordInfo 表示聊天记录信息
type RecordInfo struct {
	XMLName       xml.Name `xml:"recordinfo"`
	FromScene     string   `xml:"fromscene,omitempty"`
	FavUsername   string   `xml:"favusername,omitempty"`
	FavCreateTime string   `xml:"favcreatetime,omitempty"`
	IsChatRoom    string   `xml:"isChatRoom,omitempty"`
	Title         string   `xml:"title,omitempty"`
	Desc          string   `xml:"desc,omitempty"`
	Info          string   `xml:"info,omitempty"`
	DataList      DataList `xml:"datalist,omitempty"`
}

// DataList 表示数据列表
type DataList struct {
	Count     string     `xml:"count,attr,omitempty"`
	DataItems []DataItem `xml:"dataitem,omitempty"`
}

// DataItem 表示数据项
type DataItem struct {
	DataType      string `xml:"datatype,attr,omitempty"`
	DataID        string `xml:"dataid,attr,omitempty"`
	HTMLID        string `xml:"htmlid,attr,omitempty"`
	DataFmt       string `xml:"datafmt,omitempty"`
	SourceName    string `xml:"sourcename,omitempty"`
	SourceTime    string `xml:"sourcetime,omitempty"`
	SourceHeadURL string `xml:"sourceheadurl,omitempty"`
	DataDesc      string `xml:"datadesc,omitempty"`

	// 图片特有字段
	ThumbSourcePath  string `xml:"thumbsourcepath,omitempty"`
	ThumbSize        string `xml:"thumbsize,omitempty"`
	CDNDataURL       string `xml:"cdndataurl,omitempty"`
	CDNDataKey       string `xml:"cdndatakey,omitempty"`
	CDNThumbURL      string `xml:"cdnthumburl,omitempty"`
	CDNThumbKey      string `xml:"cdnthumbkey,omitempty"`
	DataSourcePath   string `xml:"datasourcepath,omitempty"`
	FullMD5          string `xml:"fullmd5,omitempty"`
	ThumbFullMD5     string `xml:"thumbfullmd5,omitempty"`
	ThumbHead256MD5  string `xml:"thumbhead256md5,omitempty"`
	DataSize         string `xml:"datasize,omitempty"`
	CDNEncryVer      string `xml:"cdnencryver,omitempty"`
	SrcChatname      string `xml:"srcChatname,omitempty"`
	SrcMsgLocalID    string `xml:"srcMsgLocalid,omitempty"`
	SrcMsgCreateTime string `xml:"srcMsgCreateTime,omitempty"`
	MessageUUID      string `xml:"messageuuid,omitempty"`
	FromNewMsgID     string `xml:"fromnewmsgid,omitempty"`
}
