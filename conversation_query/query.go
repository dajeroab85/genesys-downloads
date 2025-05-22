package conversation_query

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/goDownloadRecording/logger"
	sdk "github.com/mypurecloud/platform-client-sdk-go/v157/platformclientv2"
	//"go.uber.org/zap"
)

func BuildConversationQuery() sdk.Conversationquery {
	var endTime, startTime, order, orderBy, divisionId, originatingDirection string

	logger.Log.Info("Start function BuildConversationQuery")
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Ingrese fecha de Inicio (formato: yyyy-mm-ddThh:mm:ss-zz:zz):")
	if scanner.Scan() {
		startTime = scanner.Text()
	}

	fmt.Println("Ingrese fecha Fin (formato: yyyy-mm-ddThh:mm:ss-zz:zz):")
	if scanner.Scan() {
		endTime = scanner.Text()
	}

	interval := startTime + "/" + endTime

	fmt.Println("Ingrese el orden (desc / asc):")
	if scanner.Scan() {
		order = scanner.Text()
	}
	if order == "" {
		order = "desc"
	}

	fmt.Println("Ingrese el campo para ordenar (conversationStart, segmentStart, segmentEnd):")
	if scanner.Scan() {
		orderBy = scanner.Text()
	}
	if orderBy == "" {
		orderBy = "conversationStart"
	}

	fmt.Println("Ingrese el ID de la División (o deje vacío para todas):")
	if scanner.Scan() {
		divisionId = scanner.Text()
	}

	fmt.Println("¿Desea filtrar por dirección (originatingDirection)? (Ej: inbound, outbound, empty para ignorar):")
	if scanner.Scan() {
		originatingDirection = scanner.Text()
	}

	// Segment filters obligatorios
	segmentFilters := &[]sdk.Segmentdetailqueryfilter{
		{
			VarType: sdk.String("and"),
			Predicates: &[]sdk.Segmentdetailquerypredicate{
				{
					Dimension: sdk.String("mediaType"),
					Operator:  sdk.String("matches"),
					Value:     sdk.String("voice"),
				},
				{
					Dimension: sdk.String("recording"),
					Operator:  sdk.String("exists"),
				},
			},
		},
	}

	// Arma la estructura base
	query := sdk.Conversationquery{
		Order:          &order,
		OrderBy:        &orderBy,
		Interval:       &interval,
		SegmentFilters: segmentFilters,
	}

	// Filtro por división
	if divisionId != "" {
		query.ConversationFilters = &[]sdk.Conversationdetailqueryfilter{
			{
				VarType: sdk.String("or"),
				Predicates: &[]sdk.Conversationdetailquerypredicate{
					{
						Dimension: sdk.String("divisionId"),
						Operator:  sdk.String("matches"),
						Value:     &divisionId,
					},
				},
			},
		}
	}

	// Filtro por dirección de origen (optional)
	if originatingDirection != "" {
		filter := sdk.Conversationdetailqueryfilter{
			VarType: sdk.String("or"),
			Predicates: &[]sdk.Conversationdetailquerypredicate{
				{
					Dimension: sdk.String("originatingDirection"),
					Operator:  sdk.String("matches"),
					Value:     &originatingDirection,
				},
			},
		}

		// Si ya hay filtros, los agregamos
		if query.ConversationFilters != nil {
			*query.ConversationFilters = append(*query.ConversationFilters, filter)
		} else {
			query.ConversationFilters = &[]sdk.Conversationdetailqueryfilter{filter}
		}
	}

	// Mostrar JSON resultante (útil para debug)
	b, _ := json.MarshalIndent(query, "", "  ")
	fmt.Println("Consulta construida:")
	fmt.Println(string(b))

	logger.Log.Info("End function BuildConversationQuery")
	return query
}

func GetAllConversationsResults(api *sdk.AnalyticsApi, baseQuery sdk.Conversationquery) ([]sdk.Analyticsconversationwithoutattributes, error) {
	var allResults []sdk.Analyticsconversationwithoutattributes
	pageSize := 100
	pageNumber := 1

	for {
		baseQuery.Paging = &sdk.Pagingspec{
			PageSize:   &pageSize,
			PageNumber: &pageNumber,
		}

		resp, _, err := api.PostAnalyticsConversationsDetailsQuery(baseQuery)
		if err != nil {
			return nil, fmt.Errorf("error en página %d: %w", pageNumber, err)
		}

		if resp.Conversations != nil {
			allResults = append(allResults, *resp.Conversations...)
		}

		if len(*resp.Conversations) < pageSize {
			break
		}

		pageNumber++
	}

	return allResults, nil
}
