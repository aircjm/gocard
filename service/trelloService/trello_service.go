package trelloService

import (
	"github.com/adlio/trello"
	"github.com/aircjm/gocard/client"
	"github.com/aircjm/gocard/client/model"
	"github.com/aircjm/gocard/dao/ankiDao"
	"github.com/aircjm/gocard/dao/cardDao"
	"github.com/aircjm/gocard/dto"
	"github.com/aircjm/gocard/service/ankiService"
	"log"
	"time"
)

// GetRecentlyEditedCard 获取最新的卡片记录
func GetRecentlyEditedCard() (cards []*trello.Card, err error) {
	cards, err = client.TrelloCL.SearchCards("edited:week", trello.Defaults())
	return cards, err
}

// 获取包含对应标签标签的卡片
func GetLabelCard(board trello.Board, labelName string) []*trello.Card {
	cards, err := board.GetCards(trello.Defaults())
	if err != nil {
		log.Fatal(err)
	}

	var labelCardLists []*trello.Card
	for _, card := range cards {
		if len(card.Labels) == 0 {
			continue
		}
		for _, label := range card.Labels {
			if label.Name == labelName {
				labelCardLists = append(labelCardLists, card)
			}
		}
	}
	return labelCardLists
}

// GetRecentlyEditedCard 获取最新的卡片记录
func SaveRecentlyEditedCard() {
	cards, err := GetRecentlyEditedCard()
	if err != nil {
		log.Fatalln(err)
	}
	SaveCardsOrm(cards)
}

// SaveCards 批量保存cards 如果有就更新
func SaveCardsOrm(cards []*trello.Card) {
	for _, card := range cards {
		cardDao.SaveCardOrm(*card)
	}
}

func GetBoardAnkiLabelCard(board trello.Board) []*trello.Card {

	cards, err := board.GetCards(trello.Defaults())
	if err != nil {
		log.Fatal(err)
	}

	var ankiLabelCardList []*trello.Card

	for _, card := range cards {
		if len(card.Labels) == 0 {
			continue
		}
		for _, label := range card.Labels {
			if label.Name == "anki" {
				ankiLabelCardList = append(ankiLabelCardList, card)
			}
		}
	}
	return ankiLabelCardList
}

// SaveAllCards 保存所有的卡片
func SaveAllCards() {
	begin := time.Now()
	boards, err := client.TrelloCL.GetMyBoards(trello.Defaults())
	if err != nil {
		log.Fatal(err)
	}
	for _, board := range boards {

		go SaveBoard(board)
		cards, err := board.GetCards(trello.Defaults())
		if err != nil {
			log.Fatal(err)
		}
		go SaveCardsOrm(cards)
	}

	log.Println("整个方法执行的时间为：", time.Now().Sub(begin))
}

func SaveBoard(board *trello.Board) {
	cardDao.SaveBoard(*board)
}

func GetBoardList() []dto.MingBoard {
	boards := cardDao.GetBoardList()
	return boards
}

func ConvertToAnki(list []string) {
	cardList := cardDao.GetCardByCardIdList(list)
	for _, flashCard := range cardList {
		if flashCard.AnkiNoteInfo.AnkiNoteID > 0 {
			log.Println("已经有 anki note 笔记了，开始更新")
		} else {
			log.Println("新增 anki note 笔记")
			addNoteAnkiRequest := model.AnkiAddNoteRequest{}.GetAnkiAddNoteRequest(flashCard, dto.MingBoard{})

			ankiNote := ankiService.AddAnkiNote(*addNoteAnkiRequest)
			log.Println(ankiNote)
		}

	}
}

func ConvertToAnkiNote(list []string) {
	for _, cardId := range list {
		SingleConvertToAnki(cardId)
	}
}

//UpdateCardStatus 更新卡片
func UpdateCardStatus(card dto.FlashCard) {
	cardDao.UpdateCard(card)
}

func SingleConvertToAnki(cardId string) {
	var cardIdList = []string{}
	cardIdList = append(cardIdList, cardId)
	flashCards := cardDao.GetCardByCardIdList(cardIdList)
	flashCard := flashCards[0]
	if len(flashCard.Name) > 0 {
		var boardIdList []string
		boardIdList = append(boardIdList, flashCard.IDBoard)
		boards := cardDao.GetBoardListByBoardIdList(boardIdList)
		board := boards[0]
		if len(board.Name) > 0 {
			ankiNoteInfo := ankiDao.GetAnkiNoteByTrelloCardId(flashCard.ID)
			if len(ankiNoteInfo.Title) > 0 {
				if ankiNoteInfo.UpdatedAt.Before(flashCard.DateLastActivity) {
					log.Println("卡片需要更新")
				}
			} else {
				addNoteAnkiRequest := model.AnkiAddNoteRequest{}.GetAnkiAddNoteRequest(flashCard, board)
				ankiNoteId := ankiService.AddAnkiNote(*addNoteAnkiRequest)
				if ankiNoteId > 0 {
					ankiNote := dto.AnkiNoteInfo{}
					ankiNote.AnkiNoteID = ankiNoteId
					ankiNote.HtmlContext = addNoteAnkiRequest.Params.Note.Fields.Back
					ankiNote.TrelloCardId = flashCard.ID
					ankiNote.ModelName = addNoteAnkiRequest.Params.Note.ModelName
					ankiNote.DeckName = addNoteAnkiRequest.Params.Note.DeckName
					ankiNote.Status = 1
					ankiDao.SaveAnkiNote(ankiNote)
				}
			}
		}

	}

}